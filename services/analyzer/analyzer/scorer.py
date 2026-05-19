"""
Impact scorer for blast-radius analysis.

Given a set of reachable downstream nodes and a list of schema changes,
the scorer computes an impact level for each node based on field intersection.

Impact levels:
  CRITICAL: The consumer uses one or more fields that are being removed or
            type-changed. It will break when the schema change is deployed.

  HIGH:     The consumer uses fields adjacent to changed fields, or the
            change alters a required field the consumer depends on.

  MEDIUM:   The consumer is in the downstream path but does not directly
            use any changed fields. It may be affected by data quality
            changes or semantic shifts.

  NONE:     The consumer has no field overlap with the changed fields
            and is not in a critical path.

The scorer is a pure function of (changed_fields, consumer_fields, depth).
No I/O, no side effects. This makes it independently testable and reusable
across the CI gate and the catalog UI.
"""

from __future__ import annotations

from dataclasses import dataclass
from enum import Enum

from analyzer.traversal import ReachableNode


class ImpactLevel(str, Enum):
    CRITICAL = "CRITICAL"
    HIGH = "HIGH"
    MEDIUM = "MEDIUM"
    NONE = "NONE"

    @property
    def numeric(self) -> int:
        return {"CRITICAL": 4, "HIGH": 3, "MEDIUM": 2, "NONE": 1}[self.value]


@dataclass
class ScoredConsumer:
    name: str
    node_type: str
    owner_team: str | None
    impact: ImpactLevel
    affected_fields: list[str]
    depth: int
    # Direct means depth == 1 and the consumer reads the source topic.
    is_direct: bool


class ImpactScorer:
    """
    Scores downstream consumers by their exposure to a set of schema changes.

    changed_fields: field paths that are being removed, renamed, or type-changed.
    These are the fields that will cause consumer failures if the consumer
    depends on them.

    added_fields: field paths being added. Generally lower impact, but
    adding a required field is handled as CRITICAL by the evolution checker
    upstream and does not need special scoring here.
    """

    def score(
        self,
        reachable: list[ReachableNode],
        changed_fields: set[str],
        added_fields: set[str] | None = None,
    ) -> list[ScoredConsumer]:
        if not reachable:
            return []

        scored = []
        for node in reachable:
            impact, affected = self._score_node(node, changed_fields)
            scored.append(ScoredConsumer(
                name=node.name,
                node_type=node.node_type.value,
                owner_team=node.owner_team,
                impact=impact,
                affected_fields=affected,
                depth=node.depth,
                is_direct=node.depth == 1,
            ))

        # Sort by impact severity descending, then by depth ascending.
        # This puts the most critical direct consumers at the top of the report.
        scored.sort(key=lambda c: (-c.impact.numeric, c.depth))
        return scored

    def _score_node(
        self,
        node: ReachableNode,
        changed_fields: set[str],
    ) -> tuple[ImpactLevel, list[str]]:
        if not node.fields:
            # No field metadata available for this node. We cannot assess
            # field-level impact, so we assign MEDIUM for direct consumers
            # and NONE for transitive ones. This is conservative: we flag
            # the direct consumer for manual review rather than assuming safety.
            impact = ImpactLevel.MEDIUM if node.depth == 1 else ImpactLevel.NONE
            return impact, []

        # Field intersection: which of the changed fields does this consumer use?
        affected = sorted(node.fields & changed_fields)

        if affected:
            # Any overlap with removed or type-changed fields is CRITICAL.
            # The consumer will receive data it cannot process.
            impact = ImpactLevel.CRITICAL
        elif node.depth == 1:
            # Direct consumer with no field overlap: the schema change does
            # not break any fields this consumer uses, but the consumer is
            # one hop away and should be notified as a precaution.
            impact = ImpactLevel.HIGH
        else:
            impact = ImpactLevel.NONE

        return impact, affected

    def worst_impact(self, scored: list[ScoredConsumer]) -> ImpactLevel:
        if not scored:
            return ImpactLevel.NONE
        return max(scored, key=lambda c: c.impact.numeric).impact

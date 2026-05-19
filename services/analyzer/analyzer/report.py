"""
Blast-radius report assembly.

Takes the raw output of the traversal and scorer and assembles a structured
report suitable for the CI gate PR comment and the catalog UI.

The report is designed to be actionable:
- Which teams need to be notified?
- Which consumers will break immediately vs. eventually?
- What is the minimum set of approvals needed to proceed?
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone

from analyzer.scorer import ImpactLevel, ScoredConsumer


@dataclass
class BlastRadiusReport:
    topic: str
    changed_fields: list[str]
    generated_at: str
    total_consumers_affected: int
    worst_impact: ImpactLevel
    consumers: list[ScoredConsumer]
    # Teams that must approve before a CRITICAL or HIGH change is deployed.
    required_approvals: list[str] = field(default_factory=list)
    summary: str = ""

    def to_dict(self) -> dict:
        return {
            "topic": self.topic,
            "changed_fields": self.changed_fields,
            "generated_at": self.generated_at,
            "total_consumers_affected": self.total_consumers_affected,
            "worst_impact": self.worst_impact.value,
            "required_approvals": self.required_approvals,
            "summary": self.summary,
            "consumers": [
                {
                    "name": c.name,
                    "node_type": c.node_type,
                    "owner_team": c.owner_team,
                    "impact": c.impact.value,
                    "affected_fields": c.affected_fields,
                    "depth": c.depth,
                    "is_direct": c.is_direct,
                }
                for c in self.consumers
            ],
        }


class ReportAssembler:
    def assemble(
        self,
        topic: str,
        changed_fields: list[str],
        scored_consumers: list[ScoredConsumer],
    ) -> BlastRadiusReport:
        affected = [c for c in scored_consumers if c.impact != ImpactLevel.NONE]

        worst = ImpactLevel.NONE
        if affected:
            worst = max(affected, key=lambda c: c.impact.numeric).impact

        required_approvals = self._compute_required_approvals(affected, worst)

        report = BlastRadiusReport(
            topic=topic,
            changed_fields=changed_fields,
            generated_at=datetime.now(tz=timezone.utc).isoformat(),
            total_consumers_affected=len(affected),
            worst_impact=worst,
            consumers=affected,
            required_approvals=required_approvals,
            summary=self._build_summary(topic, changed_fields, affected, worst),
        )

        return report

    def _compute_required_approvals(
        self,
        affected: list[ScoredConsumer],
        worst: ImpactLevel,
    ) -> list[str]:
        if worst not in (ImpactLevel.CRITICAL, ImpactLevel.HIGH):
            return []

        teams: set[str] = set()
        for consumer in affected:
            if consumer.impact in (ImpactLevel.CRITICAL, ImpactLevel.HIGH):
                if consumer.owner_team:
                    teams.add(consumer.owner_team)

        return sorted(teams)

    def _build_summary(
        self,
        topic: str,
        changed_fields: list[str],
        affected: list[ScoredConsumer],
        worst: ImpactLevel,
    ) -> str:
        if not affected:
            return (
                f"No downstream consumers are impacted by changes to "
                f"{', '.join(changed_fields)} in {topic}."
            )

        critical = [c for c in affected if c.impact == ImpactLevel.CRITICAL]
        high = [c for c in affected if c.impact == ImpactLevel.HIGH]
        direct = [c for c in affected if c.is_direct]

        parts = [
            f"Changing {', '.join(changed_fields)} in {topic} affects "
            f"{len(affected)} downstream consumer(s)."
        ]

        if critical:
            names = ", ".join(c.name for c in critical[:3])
            suffix = f" and {len(critical) - 3} more" if len(critical) > 3 else ""
            parts.append(f"{len(critical)} CRITICAL: {names}{suffix} will break.")

        if high:
            parts.append(f"{len(high)} HIGH impact consumer(s) require notification.")

        if direct:
            parts.append(f"{len(direct)} consumer(s) read this topic directly.")

        return " ".join(parts)

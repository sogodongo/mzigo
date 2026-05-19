import pytest

from analyzer.graph import NodeType
from analyzer.scorer import ImpactLevel, ImpactScorer
from analyzer.traversal import ReachableNode


def make_node(
    name: str,
    fields: set[str],
    depth: int = 1,
    owner_team: str | None = "some-team",
) -> ReachableNode:
    return ReachableNode(
        node_id=f"job:mzigo:{name}",
        name=name,
        node_type=NodeType.JOB,
        owner_team=owner_team,
        fields=fields,
        depth=depth,
        path=[f"job:mzigo:{name}"],
    )


@pytest.fixture
def scorer() -> ImpactScorer:
    return ImpactScorer()


def test_consumer_using_removed_field_is_critical(scorer):
    consumers = [make_node("fraud-flink", fields={"transaction_id", "amount_cents"})]
    changed = {"amount_cents"}

    scored = scorer.score(consumers, changed)

    assert len(scored) == 1
    assert scored[0].impact == ImpactLevel.CRITICAL
    assert "amount_cents" in scored[0].affected_fields


def test_consumer_not_using_changed_field_is_high_if_direct(scorer):
    consumers = [make_node("analytics-flink", fields={"transaction_id", "currency"})]
    changed = {"amount_cents"}

    scored = scorer.score(consumers, changed)

    assert scored[0].impact == ImpactLevel.HIGH
    assert scored[0].affected_fields == []


def test_transitive_consumer_with_no_overlap_is_none(scorer):
    consumers = [make_node("downstream-sink", fields={"transaction_id"}, depth=3)]
    changed = {"amount_cents"}

    scored = scorer.score(consumers, changed)

    assert scored[0].impact == ImpactLevel.NONE


def test_consumer_with_no_field_metadata_is_medium_if_direct(scorer):
    consumers = [make_node("unknown-consumer", fields=set(), depth=1)]
    changed = {"amount_cents"}

    scored = scorer.score(consumers, changed)

    assert scored[0].impact == ImpactLevel.MEDIUM


def test_consumer_with_no_field_metadata_is_none_if_transitive(scorer):
    consumers = [make_node("deep-consumer", fields=set(), depth=3)]
    changed = {"amount_cents"}

    scored = scorer.score(consumers, changed)

    assert scored[0].impact == ImpactLevel.NONE


def test_results_sorted_by_severity_then_depth(scorer):
    consumers = [
        make_node("high-depth-2", fields={"transaction_id"}, depth=2),
        make_node("critical-depth-1", fields={"amount_cents"}, depth=1),
        make_node("high-depth-1", fields={"currency"}, depth=1),
    ]
    changed = {"amount_cents"}

    scored = scorer.score(consumers, changed)

    assert scored[0].name == "critical-depth-1"
    assert scored[0].impact == ImpactLevel.CRITICAL


def test_worst_impact_returns_highest_severity(scorer):
    consumers = [
        make_node("a", fields={"currency"}, depth=1),
        make_node("b", fields={"amount_cents"}, depth=1),
    ]
    changed = {"amount_cents"}

    scored = scorer.score(consumers, changed)
    worst = scorer.worst_impact(scored)

    assert worst == ImpactLevel.CRITICAL


def test_empty_consumers_returns_none_impact(scorer):
    assert scorer.worst_impact([]) == ImpactLevel.NONE

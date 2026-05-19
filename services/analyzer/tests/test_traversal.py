"""
Traversal tests use synthetic NetworkX graphs rather than a live Marquez instance.
This keeps the tests fast, deterministic, and free of external dependencies.

The graph structure we build here mirrors what Marquez returns for a
two-hop lineage path: topic -> flink job -> iceberg table.
"""

import networkx as nx
import pytest

from analyzer.graph import NodeType
from analyzer.traversal import DAGTraverser


def build_test_graph() -> nx.DiGraph:
    """
    topic:mzigo:payments.transactions
        -> job:mzigo:fraud-detection-flink
            -> dataset:mzigo:iceberg.fraud_signals
        -> job:mzigo:ledger-reconciliation
            -> dataset:mzigo:iceberg.ledger_entries
    """
    g = nx.DiGraph()

    g.add_node(
        "dataset:mzigo:payments.transactions",
        node_type=NodeType.DATASET,
        name="payments.transactions",
        namespace="mzigo",
        owner_team="payments-platform",
        fields={"transaction_id", "amount_cents", "currency", "account_id", "status"},
    )
    g.add_node(
        "job:mzigo:fraud-detection-flink",
        node_type=NodeType.JOB,
        name="fraud-detection-flink",
        namespace="mzigo",
        owner_team="fraud-engineering",
        fields={"transaction_id", "amount_cents", "account_id"},
    )
    g.add_node(
        "dataset:mzigo:iceberg.fraud_signals",
        node_type=NodeType.DATASET,
        name="iceberg.fraud_signals",
        namespace="mzigo",
        owner_team="fraud-engineering",
        fields={"transaction_id", "risk_score"},
    )
    g.add_node(
        "job:mzigo:ledger-reconciliation",
        node_type=NodeType.JOB,
        name="ledger-reconciliation",
        namespace="mzigo",
        owner_team="finance-engineering",
        fields={"transaction_id", "amount_cents", "currency", "status"},
    )
    g.add_node(
        "dataset:mzigo:iceberg.ledger_entries",
        node_type=NodeType.DATASET,
        name="iceberg.ledger_entries",
        namespace="mzigo",
        owner_team="finance-engineering",
        fields={"transaction_id", "amount_cents"},
    )

    g.add_edge("dataset:mzigo:payments.transactions", "job:mzigo:fraud-detection-flink")
    g.add_edge("job:mzigo:fraud-detection-flink", "dataset:mzigo:iceberg.fraud_signals")
    g.add_edge("dataset:mzigo:payments.transactions", "job:mzigo:ledger-reconciliation")
    g.add_edge("job:mzigo:ledger-reconciliation", "dataset:mzigo:iceberg.ledger_entries")

    return g


@pytest.fixture
def graph() -> nx.DiGraph:
    return build_test_graph()


@pytest.fixture
def traverser() -> DAGTraverser:
    return DAGTraverser()


def test_finds_direct_consumers(graph, traverser):
    reachable = traverser.find_reachable(
        graph,
        "dataset:mzigo:payments.transactions",
        max_depth=5,
    )
    names = {n.name for n in reachable}
    assert "fraud-detection-flink" in names
    assert "ledger-reconciliation" in names


def test_finds_transitive_consumers(graph, traverser):
    reachable = traverser.find_reachable(
        graph,
        "dataset:mzigo:payments.transactions",
        max_depth=5,
    )
    names = {n.name for n in reachable}
    assert "iceberg.fraud_signals" in names
    assert "iceberg.ledger_entries" in names


def test_direct_consumers_have_depth_one(graph, traverser):
    reachable = traverser.find_reachable(
        graph,
        "dataset:mzigo:payments.transactions",
        max_depth=5,
    )
    direct = {n.name: n.depth for n in reachable if n.depth == 1}
    assert "fraud-detection-flink" in direct
    assert "ledger-reconciliation" in direct


def test_transitive_consumers_have_depth_two(graph, traverser):
    reachable = traverser.find_reachable(
        graph,
        "dataset:mzigo:payments.transactions",
        max_depth=5,
    )
    transitive = {n.name: n.depth for n in reachable if n.depth == 2}
    assert "iceberg.fraud_signals" in transitive
    assert "iceberg.ledger_entries" in transitive


def test_max_depth_limits_traversal(graph, traverser):
    reachable = traverser.find_reachable(
        graph,
        "dataset:mzigo:payments.transactions",
        max_depth=1,
    )
    names = {n.name for n in reachable}
    # With max_depth=1, only direct consumers are found
    assert "fraud-detection-flink" in names
    assert "iceberg.fraud_signals" not in names


def test_unknown_source_returns_empty(graph, traverser):
    reachable = traverser.find_reachable(graph, "dataset:mzigo:nonexistent", max_depth=5)
    assert reachable == []


def test_find_source_node_id(graph, traverser):
    node_id = traverser.find_source_node_id(graph, "payments.transactions", "mzigo")
    assert node_id == "dataset:mzigo:payments.transactions"


def test_find_source_node_id_missing(graph, traverser):
    node_id = traverser.find_source_node_id(graph, "unknown.topic", "mzigo")
    assert node_id is None

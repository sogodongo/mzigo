"""
DAG traversal for blast-radius analysis.

Given a lineage graph rooted at a source topic, we need to find all
downstream nodes that are reachable from that root and classify each
as a direct or transitive consumer.

Direct consumers: one hop from the source topic (Flink jobs, consumers
reading the topic directly).

Transitive consumers: two or more hops away. A Flink job reads topic A,
writes to topic B, and topic B is read by an Iceberg sink. The Iceberg
sink is a transitive consumer of topic A's schema.

We distinguish these because impact severity differs:
- Direct consumers break immediately when a schema change is deployed.
- Transitive consumers break when the intermediate job propagates the change,
  which may be delayed by minutes or hours depending on job checkpointing.
"""

from __future__ import annotations

from dataclasses import dataclass

import networkx as nx
import structlog

from analyzer.graph import NodeType

log = structlog.get_logger(__name__)


@dataclass
class ReachableNode:
    node_id: str
    name: str
    node_type: NodeType
    owner_team: str | None
    fields: set[str]
    # Hop count from the source topic. 1 = direct consumer.
    depth: int
    # Path from source to this node, for trace/debugging in the report.
    path: list[str]


class DAGTraverser:
    """
    Traverses a lineage DAG to find all nodes reachable from a source.

    We use BFS rather than DFS because BFS naturally gives us the
    shortest path to each node, which maps directly to the "minimum
    depth at which a consumer is affected" concept.
    """

    def find_reachable(
        self,
        graph: nx.DiGraph,
        source_node_id: str,
        max_depth: int,
    ) -> list[ReachableNode]:
        """
        Returns all nodes reachable from source_node_id within max_depth hops,
        excluding the source node itself.
        """
        if source_node_id not in graph:
            log.warning("source_node_not_in_graph", node_id=source_node_id)
            return []

        reachable: list[ReachableNode] = []
        visited: set[str] = {source_node_id}

        # BFS queue: (node_id, depth, path_so_far)
        queue: list[tuple[str, int, list[str]]] = [
            (source_node_id, 0, [source_node_id])
        ]

        while queue:
            current_id, depth, path = queue.pop(0)

            if depth >= max_depth:
                continue

            for neighbor_id in graph.successors(current_id):
                if neighbor_id in visited:
                    continue

                visited.add(neighbor_id)
                node_attrs = graph.nodes.get(neighbor_id, {})
                neighbor_path = [*path, neighbor_id]

                reachable.append(ReachableNode(
                    node_id=neighbor_id,
                    name=node_attrs.get("name", neighbor_id),
                    node_type=node_attrs.get("node_type", NodeType.DATASET),
                    owner_team=node_attrs.get("owner_team"),
                    fields=node_attrs.get("fields", set()),
                    depth=depth + 1,
                    path=neighbor_path,
                ))

                queue.append((neighbor_id, depth + 1, neighbor_path))

        return reachable

    def find_source_node_id(self, graph: nx.DiGraph, topic: str, namespace: str) -> str | None:
        """
        Finds the graph node ID that corresponds to a Kafka topic name.
        Marquez node IDs are formatted as "dataset:{namespace}:{name}".
        """
        canonical = f"dataset:{namespace}:{topic}"
        if canonical in graph:
            return canonical

        # Fall back to searching by name attribute for non-canonical IDs
        for node_id, attrs in graph.nodes(data=True):
            if attrs.get("name") == topic:
                return node_id

        return None

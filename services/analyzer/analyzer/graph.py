"""
Lineage DAG construction.

Queries Marquez to build a directed graph of dataset and job relationships.
The graph is the foundation for blast-radius traversal.

Marquez models lineage as:
  Dataset --[read by]--> Job --[writes to]--> Dataset

We translate this into a NetworkX DiGraph where:
  Nodes: datasets (Kafka topics, Iceberg tables) and jobs (Flink jobs, producers)
  Edges: directed from input dataset to job, and from job to output dataset

Node attributes carry the metadata needed for impact scoring:
  - node_type: DATASET or JOB
  - dataset_type: KAFKA_TOPIC, ICEBERG_TABLE, UNKNOWN
  - owner_team: team responsible for this node (from Marquez facets)
  - fields: set of field names present in this dataset
"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum

import httpx
import networkx as nx
import structlog

log = structlog.get_logger(__name__)


class NodeType(str, Enum):
    DATASET = "DATASET"
    JOB = "JOB"


class DatasetType(str, Enum):
    KAFKA_TOPIC = "KAFKA_TOPIC"
    ICEBERG_TABLE = "ICEBERG_TABLE"
    UNKNOWN = "UNKNOWN"


@dataclass
class LineageNode:
    name: str
    node_type: NodeType
    namespace: str
    dataset_type: DatasetType = DatasetType.UNKNOWN
    owner_team: str | None = None
    fields: set[str] = field(default_factory=set)


class LineageGraphBuilder:
    """
    Fetches lineage from Marquez and constructs a NetworkX DiGraph.

    We use Marquez's /api/v1/lineage endpoint which returns a subgraph
    rooted at a given dataset. The depth parameter controls how many
    hops to follow from the root.
    """

    def __init__(self, marquez_url: str, namespace: str, max_depth: int) -> None:
        self._marquez_url = marquez_url.rstrip("/")
        self._namespace = namespace
        self._max_depth = max_depth
        self._client = httpx.Client(timeout=30.0)

    def build_for_topic(self, topic: str) -> nx.DiGraph:
        """
        Build a lineage DAG rooted at the given Kafka topic.
        Returns an empty graph if Marquez has no lineage for this topic.
        """
        graph = nx.DiGraph()

        try:
            data = self._fetch_lineage(topic)
        except httpx.HTTPError as exc:
            log.warning("marquez_fetch_failed", topic=topic, error=str(exc))
            return graph

        if not data:
            return graph

        self._populate_graph(graph, data)
        return graph

    def _fetch_lineage(self, topic: str) -> dict:
        url = (
            f"{self._marquez_url}/api/v1/lineage"
            f"?nodeId=dataset:{self._namespace}:{topic}"
            f"&depth={self._max_depth}"
        )
        response = self._client.get(url)
        response.raise_for_status()
        return response.json()

    def _populate_graph(self, graph: nx.DiGraph, data: dict) -> None:
        # Marquez returns a graph with "graph" key containing nodes and edges.
        nodes_raw = data.get("graph", [])

        for node_data in nodes_raw:
            node_id = node_data.get("id", "")
            node_type_raw = node_data.get("type", "")

            if node_type_raw == "DATASET":
                self._add_dataset_node(graph, node_id, node_data)
            elif node_type_raw == "JOB":
                self._add_job_node(graph, node_id, node_data)

            # Add edges from in-edges and out-edges
            for in_id in node_data.get("inEdges", []):
                graph.add_edge(in_id.get("origin"), node_id)
            for out_id in node_data.get("outEdges", []):
                graph.add_edge(node_id, out_id.get("destination"))

    def _add_dataset_node(self, graph: nx.DiGraph, node_id: str, data: dict) -> None:
        name = data.get("data", {}).get("name", node_id)
        facets = data.get("data", {}).get("facets", {})

        fields: set[str] = set()
        schema_facet = facets.get("schema", {})
        for f in schema_facet.get("fields", []):
            if fname := f.get("name"):
                fields.add(fname)

        owner_team = facets.get("ownership", {}).get("owners", [{}])[0].get("name")

        graph.add_node(
            node_id,
            node_type=NodeType.DATASET,
            name=name,
            namespace=self._namespace,
            dataset_type=self._classify_dataset(name),
            owner_team=owner_team,
            fields=fields,
        )

    def _add_job_node(self, graph: nx.DiGraph, node_id: str, data: dict) -> None:
        name = data.get("data", {}).get("name", node_id)
        facets = data.get("data", {}).get("facets", {})
        owner_team = facets.get("ownership", {}).get("owners", [{}])[0].get("name")

        graph.add_node(
            node_id,
            node_type=NodeType.JOB,
            name=name,
            namespace=self._namespace,
            owner_team=owner_team,
            fields=set(),
        )

    @staticmethod
    def _classify_dataset(name: str) -> DatasetType:
        # Heuristic classification based on naming conventions.
        # Production deployments can override via Marquez dataset facets.
        if "." in name and not name.startswith("db."):
            return DatasetType.KAFKA_TOPIC
        if name.startswith("iceberg.") or name.endswith("_table"):
            return DatasetType.ICEBERG_TABLE
        return DatasetType.UNKNOWN

    def close(self) -> None:
        self._client.close()

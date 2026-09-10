# ADR-002: Graph Representation for Topology and Algorithms

## Status

Accepted

## Context

Atlas models infrastructure (nodes, services, dependencies) as a graph and needs to run
graph algorithms against that model: BFS/DFS traversal, shortest paths (Dijkstra, A*),
cycle detection, topological sorting, connected components, Tarjan SCC, articulation
points, and bridges.

## Decision

Use an adjacency list representation (`map[string][]Edge`) as the canonical graph model
in `internal/algorithms/graph`.

- Nodes are keyed by string ID; edges carry a source ID, target ID, weight, and metadata.
- Graphs may be directed or undirected (`Directed` flag).
- The topology engine wraps a directed graph whose edges represent `depends_on`,
  `communicates_with`, `hosted_on`, `routes_to`, `replicates_to`, and `connects_to`
  relationships.

Connectivity-based algorithms (articulation points, bridges) treat the graph as undirected
regardless of the directed flag, because their definition assumes undirected connectivity.
This keeps results well-defined for the topology dependency graph.

## Alternatives Considered

- **Adjacency matrix**: O(V²) memory, poor for sparse real-world topologies.
- **Edge list**: simple but slow for neighbor queries.
- **Object graph (node structs holding pointers)**: convenient in OO languages but
  complicates serialization and ID-based lookups.

## Tradeoffs

**Chosen:**
- O(V + E) neighbor iteration, O(1) node lookup by ID.
- Natural fit for string-keyed infrastructure IDs.

**Consequences:**
- Thin wrapper types needed for typed nodes/edges (satisfied by `internal/models`).
- Edge-list operations (e.g., finding all edges targeting a node) are O(E); the topology
  engine accepts this because E is bounded at the scale targeted here.

## Consequences

Provides a single graph engine shared by topology analysis, dependency impact analysis,
recovery path computation, and chaos experiment planning.
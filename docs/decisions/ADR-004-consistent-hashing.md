# ADR-004: Consistent Hashing with Virtual Nodes

## Status

Accepted

## Context

Atlas must distribute keys (jobs, KV entries) across a changing set of nodes with minimal
remapping when nodes join or leave. Naive modulo hashing remaps almost every key on any
membership change.

## Decision

Use consistent hashing with 150 virtual nodes per physical node in
`internal/algorithms/hashing`.

- `HashRing` maps `crc32("node#vn")` to a physical node and keeps the ring sorted.
- `GetNode(key)` performs binary search on the ring: O(log n).
- `GetNodes(key, count)` walks the ring forward to support replication.
- `Distribution` and `RemapCount` expose remapping metrics for evaluation.

The job queue and KV store both route/own data through the same ring so that ownership is
coherent across subsystems.

## Alternatives Considered

- **Modulo hashing**: simple but catastrophic remapping on membership change.
- **Rendezvous (HRW) hashing**: great distribution but O(n) lookup per key.
- **Jump hashing**: O(log n) and minimal remap but only ordered node sets, no replication
  ordering.

## Tradeoffs

**Chosen:**
- O(log n) lookups, minimal remapping, natural replication ordering.

**Consequences:**
- 150 virtual nodes multiplied by the node count consumes memory; acceptable at cluster
  scale targeted by Atlas.
- `crc32` is not cryptographically strong; sufficient for distribution, not adversarial use.

## Consequences

Consistent ownership enables the sharded job queue (`ShardedQueue`) and KV replication to
share one distribution model and to behave predictably under worker churn.
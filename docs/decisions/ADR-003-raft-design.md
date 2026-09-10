# ADR-003: Raft Consensus Design for Atlas

## Status

Accepted

## Context

Atlas needs a leader-elected, replicated component for cluster state so that at most one
control plane node owns writes at any time, and the system continues operating when a
control plane node fails.

## Decision

Implement Raft from scratch in `internal/consensus/raft` (educational scope), covering:

- Leader election with randomized election timeouts (follower, candidate, leader states).
- Terms, `votedFor` per term, and log-up-to-date vote rules.
- Heartbeats implemented as AppendEntries with no entries.
- Log replication with consistency checks (`prevLogIndex`/`prevLogTerm`) and
  leader-driven `nextIndex`/`matchIndex` tracking.
- Leader failure detection via heartbeat timeout and re-election.

Messaging between nodes uses a `Transport` interface; tests use an in-memory channel
transport. Thread safety is provided by a single mutex guarding the node's state and a
goroutine-per-node main loop that drains the mailbox.

## Alternatives Considered

- **etcd/raft library**: production-grade but adds heavy dependency and hides internals.
- **CRDT-based replication**: simpler under partitions but complicates strong ordering.
- **Leaderless quorum writes**: more complex to reason about for a portfolio project.

## Tradeoffs

**Chosen:**
- Clear, auditable implementation that demonstrates understanding of consensus.

**Consequences:**
- Educational implementation is not battle-tested at scale; production rollout would
  layer it over a persistent log and network transport.
- Single-mutex design limits throughput; acceptable for control-plane workloads.
- Two key correctness fixes were required during development:
  1. Vote counting must happen in a goroutine so the mailbox loop keeps processing
     `request_vote_response` messages (a blocking election loop starves its own votes).
  2. Broadcasts must snapshot state under one lock to avoid nested RLock deadlock.

## Consequences

Raft-backed cluster coordination composes with the job queue and KV store so a single
Atlas control plane can serialize writes across replicas.
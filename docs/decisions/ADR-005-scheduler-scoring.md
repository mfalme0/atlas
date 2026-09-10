# ADR-005: Scheduler Scoring Model

## Status

Accepted

## Context

Atlas schedules workloads ("jobs") onto registered nodes. The scheduler must choose the
best-fit node and explain its decision, so operators and the UI can audit placements.

## Decision

Use a weighted additive scoring model in `internal/scheduler`, higher score wins:

- Resource fit scores for CPU, memory, and GPU: `min(available / required, 1.0)` when a
  requirement exists, 1.0 otherwise.
- Network score that decays toward 0 as actual latency approaches the workload's
  configured `LatencyMax`.
- Health score from the target node's health metric (0-1).
- Affinity score: fraction of matched affinity labels (0-1); an anti-affinity match
  yields -1.0.
- Utilization penalty: linear penalty once the node's average CPU/memory utilization
  exceeds 0.8, to discourage pile-up.
- Priority bonus scaled by priority / 100.

`ScheduleAll` updates each chosen node's utilization so multi-job batches pack realistically.
The event bus receives a `job.assigned` event for every placement.

## Alternatives Considered

- **Single-dimension bin packing**: ignores health, affinity, and explanation.
- **Pure constraint solver**: expressive but heavy and hard to explain.
- **Random/round-robin**: trivially fair but ignores fitness.

## Tradeoffs

**Chosen:**
- Transparent: every placement carries a `ScoreBreakdown` with per-axis contributions.
- Configurable weights can be layered on top without changing the model.

**Consequences:**
- Additive model can admit balanced rather than globally optimal packs; acceptable for the
  targeted use case and better for explainability.
- Tuning (weights, penalty thresholds) remains a future iteration.

## Consequences

The job queue integrates the scheduler by scoring each submitted job against registered
nodes and recording the winning node as the job's `AssignedNode`.
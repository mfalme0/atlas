# ADR-008: Incident Correlation and Advisory Analysis

## Status
Accepted

## Context
Atlas receives a noisy stream of signals: metric anomalies, node failures,
job failures, and chaos experiments. Before any analysis is useful, signals
must be correlated into incidents, and incidents must have a lifecycle
(open -> investigating -> resolved). M11 also introduces an "AI" analysis
component whose scope was constrained by ADR-007 (read-only, advisory only,
human-in-the-loop).

## Decision

### Incident correlation (internal/incident)
- Signals of interest: `anomaly.detected`, `node.failed`, `job.failed`,
  `chaos.started` (consumed from the event bus plus a direct `RecordSignal`
  path for tests and non-bus producers).
- Single-node coalescing: signals on the same node within the correlation
  window (default 60s) merge into that node's open incident and escalate its
  severity if a critical signal arrives.
- Cluster-wide correlation: when the same metric is anomalous on at least
  `ClusterThreshold` (default 3) distinct nodes within the window, a
  cluster-wide incident is opened, and the correlated single-node incidents
  are absorbed (resolved with "superseded by cluster-wide correlation"). The
  count includes the arriving signal's node, which has not yet been attached.
- Incidents are views (`incident.View`) carrying affected nodes, signals,
  lifecycle state, and analysis output.

### Advisory analysis (internal/ai)
- `ai.Advisor.Analyze` is deterministic and strictly read-only. It consumes a
  `incident.View`, its signals, and a read-only topology `Snapshot` interface
  (`GetNode`, `HealthScore`, `Bottlenecks`).
- Classification archetypes, checked in priority order:
  deliberate_failure_injection (an active chaos experiment is the dominant
  explanation), dependency_cascade (2+ failed components),
  single_component_failure, cluster_wide_pressure (same metric, N nodes),
  capacity_bottleneck (degraded nodes), latency_degradation, indeterminate.
- Every `Recommendation` carries `RequiresApproval: true`. No code path in
  the package mutates nodes, jobs, or cluster state (ADR-007).
- The advisor is injected via the topology engine which already satisfies the
  `Snapshot` interface; no new coupling between packages was required.

## Consequences
- Incidents become the single correlation-and-escalation surface the API and
  future tooling operate on; single-source titles and severities are
  deterministic.
- Cluster-wide correlation supersedes single-node incidents rather than
  duplicating them, keeping the open-incident list stable.
- AI output is a hypothesis plus advisories: humans always trigger
  remediations, which keeps the safety boundary testable and auditable.
- Deterministic classification means the same signals always produce the same
  archetype, making the advisor unit-testable in the CI pipeline without a
  model or network dependency.
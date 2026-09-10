# ADR-007: AI Analysis Safety Boundary

## Status

Accepted

## Context

Atlas plans to generate incident analysis and root-cause hypotheses using AI. The AI
component must never be able to mutate cluster state autonomously: a nondeterministic
model amplifying a wrong hypothesis into an automated action would be dangerous.

## Decision

Apply a strict safety boundary between the AI analysis layer and the acting layer:

1. **Read-only inputs**: the AI analysis adapter consumes snapshots (topology snapshot,
   incident details, metrics) and never receives write-capable handles.
2. **Advisory-only outputs**: AI output is stored as a free-text `AIAnalysis` field on an
   incident and rendered to operators. It is never executed as a command.
3. **Human-in-the-loop**: any remediation derived from AI output must be approved and
   submitted through the explicit job queue / chaos engine API by an operator.
4. **Auditability**: every AI-generated analysis carries the analyst model ID, the input
   snapshot timestamp, and a confidence assessment in the incident record.

## Alternatives Considered

- **Autonomous remediation**: riskier and undesirable for this project's scope.
- **No AI layer at all**: avoids the concern but forfeits a differentiating feature.

## Tradeoffs

**Chosen:**
- Unbounded accuracy expectations are bounded by an explicit audit trail.

**Consequences:**
- AI analysis is a presentation/insight layer only; remediation latency depends on a
  human (or an explicitly configured, well-scoped automation policy) in the loop.

## Consequences

Guides implementation of the incident analysis milestone: the analyst will be a passive
consumer of the event bus, writing evidence-backed suggestions to the incident store
without any code path that mutates cluster resources.
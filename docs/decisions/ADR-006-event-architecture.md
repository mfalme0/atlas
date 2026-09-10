# ADR-006: Event-Driven Architecture for Cluster Notifications

## Status

Accepted

## Context

Atlas subsystems (topology, scheduler, job queue, future chaos engine, anomaly detection)
need to react to state changes without tight coupling. Emission points are numerous and
cross-cutting (node discovery, job lifecycle, leader election, incidents).

## Decision

Use a lightweight in-process pub/sub event bus in `internal/event`.

- Publishers call `bus.Publish(event.NewEvent(type, source, data))`.
- Subscribers register handlers per `EventType`.
- The bus copies events into a bounded channel; a dispatcher goroutine fans each event
  out to subscribed handlers in their own goroutines.
- Publish never blocks the caller (drops only when the buffer is full).

Event types cover the full lifecycle: node discovered/removed/failed/recovered, service
started/stopped, job created/assigned/completed/failed/retried, leader elected/failed,
anomaly detected, incident created, and chaos experiment lifecycle.

## Alternatives Considered

- **Direct method calls between subsystems**: simple but couples modules and makes
  observability harder.
- **External message broker**: adds operational complexity; not needed for a single-process
  control plane.

## Tradeoffs

**Chosen:**
- Decoupled, easily observable, and testable with in-process handler registration.

**Consequences:**
- In-process bus does not survive process restart; durable fan-out is deferred to the
  job queue / database layer.
- Blocking publish is avoided, so backpressure is implicit (drop oldest) rather than
  explicit.

## Consequences

Provides the substrate for cross-cutting features (e.g., chaos experiments can observe
topology changes, anomaly detection can subscribe to node failure events) and for the
API/UI to stream cluster activity.
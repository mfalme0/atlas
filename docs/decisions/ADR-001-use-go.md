# ADR-001: Use Go as the Primary Backend Language

## Status

Accepted

## Context

Atlas requires a backend language that supports:
- High concurrency for distributed systems
- Strong standard library for networking and serialization
- Fast compilation and deployment
- Industry adoption for infrastructure tooling

## Decision

Use Go (1.27+) as the primary backend language.

## Alternatives Considered

- **Rust**: Memory safety without GC, but steeper learning curve and slower development velocity for a portfolio project.
- **Java/JVM**: Strong ecosystem but heavier runtime and more verbose code.
- **Python**: Rapid prototyping but poor concurrency primitives and performance.

## Tradeoffs

**Chosen:**
- Goroutines and channels provide natural concurrency model
- Standard library covers HTTP, RPC, and serialization
- Fast compilation for rapid iteration
- Industry standard for infrastructure tools (Docker, Kubernetes, Terraform, etcd)

**Consequences:**
- GC pauses in extreme low-latency scenarios (acceptable for this use case)
- Manual memory management discipline still needed for high-performance components
- Must implement data structures from scratch (educational benefit for this project)

## Consequences

This decision aligns with Atlas's goals as a systems engineering portfolio project. Go is the dominant language in infrastructure tooling, and using it demonstrates relevant industry skills.

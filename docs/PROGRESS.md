# Atlas — Progress Report

Status: **all planned milestones M0–M15 implemented and passing**
Last verified: `go vet ./...` clean, `go build ./...` clean, `go test ./... -count=1` all green.

## Milestone Status

| # | Milestone | Status | Notes |
|---|-----------|--------|-------|
| M0 | Project scaffold, module, CI, README | Done | Go 1.27 module `github.com/atlas-engine/atlas`; GitHub Actions (ubuntu-latest, race detection in CI) |
| M1 | Data structures & algorithms | Done | `internal/algorithms/{array, linkedlist, stack, queue, heap, hashtable, tree (BST+AVL), trie, cache (LRU), graph}` — hashtable added to complete the set |
| M2 | Consistent hashing | Done | `internal/algorithms/hashing`, virtual nodes, tested |
| M3 | Graph engine | Done | adjacency representation, directed/undirected, BFS/DFS, Dijkstra, topological sort, articulation points + bridges (deterministic output) |
| M4 | Topology engine | Done | node/edge registry, health scores, API surface |
| M5 | Scheduler | Done | resource scoring with headroom bonus, health/affinity/priority weighting, deterministic tie-break (see ADR-005) |
| M6 | KV store | Done | key/value service on the event bus |
| M7 | Raft consensus | Done | elections, log replication, snapshots; non-blocking vote counting (ADR-003) |
| M8 | Lock-free job queue | Done | priority queue, retries, events |
| M9 | Chaos engineering | Done | HTTP API (`POST /api/v1/chaos`, active/stop), engine auto-reverts experiments, includes fault types (kill_node/failure/memory_leak/...) |
| M10 | Anomaly detection | Done | rolling-window z-score detector + absolute rules; `POST /api/v1/anomalies/ingest`, rules, list |
| M11 | Incident manager + AI advisory | Done | event correlation, clustering, lifecycle API; AI advisor returns read-only recommendations with `RequiresApproval` (ADR-007, ADR-008) |
| M12 | Observability | Done | Prometheus text + JSON metrics endpoints, event bus counters, runtime collector |
| M13 | CLI | Done | `cmd/atlas-cli`: nodes, jobs, chaos, incidents, anomalies, metrics, benchmarks; verified end-to-end against a live server |
| M14 | Web dashboard | Done | Next.js dashboard with topology/intents/anomalies/chaos/jobs/nodes; `npm run build` + `npm run lint` clean |
| M15 | Benchmark runner | Done | in-process runner over every algorithm with recorded `BenchmarkResult`, `GET/POST /api/v1/benchmarks` API + CLI |

## Architecture

- `cmd/atlas-server` — HTTP API server; engines compose via an event bus (`internal/event`)
- `cmd/atlas-cli` — operational client (default `http://localhost:8080`, env `ATLAS_API_URL`)
- `web/` — Next.js static dashboard (default `http://localhost:8080/api/v1`, env `NEXT_PUBLIC_ATLAS_API_URL`)
- `internal/api` — chi router; deps wired from main (topology, job queue, chaos, anomaly, incident, AI, metrics, benchmarks)
- `docs/decisions/ADR-001..008` — decision trail

## Verification

- Unit tests: every internal package (`go test ./... ...` all `ok`), including heavy suites (raft, benchmarks ~2 min real runs).
- API integration tests: lifecycle tests for chaos, incidents+AI analysis, anomalies, metrics, benchmarks (`internal/api/server_test.go`).
- End-to-end smoke (live server + CLI): nodes add, jobs create, chaos run -> auto-open incident, anomaly ingest -> detected, metrics counters render, incident analyze -> archetype + confidence, benchmarks run.
- Static: `go vet ./...` clean; `go build ./...` clean; both binaries produced in `build/`.

## Known Limitations

- `-race` is not runnable locally on Windows (no gcc/cgo); the CI workflow runs race tests on ubuntu-latest.
- Windows Application Control can intermittently block freshly compiled test binaries; workaround is to run via `cmd /c` or retry. Not an issue for CI.
- AI advisory strictly returns recommendations with `RequiresApproval: true`; no automated remediation is wired anywhere (by design, ADR-007).
- Benchmark numbers are measured on the local machine; treat as relative indicators, not production capacity figures.

## How to Run

```sh
# server (listens on :8080)
go run ./cmd/atlas-server

# CLI
go run ./cmd/atlas-cli status
go run ./cmd/atlas-cli nodes add --id db-1 --cpu 8 --memory 16
go run ./cmd/atlas-cli chaos run --type kill_node --target db-1 --duration 30s
go run ./cmd/atlas-cli incidents --open
go run ./cmd/atlas-cli benchmarks run all

# web dashboard
cd web && npm install && npm run dev
```
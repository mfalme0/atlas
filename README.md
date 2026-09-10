# Atlas

**An open-source distributed infrastructure intelligence engine and systems engineering laboratory.**

Atlas discovers and models infrastructure as a graph, analyzes dependencies, schedules workloads,
injects controlled failures, correlates those failures into incidents, and produces advisory AI
analysis — with real-time visualization and algorithmic transparency.

Everything is implemented from scratch: the data structures, the graph algorithms, the scheduler,
the Raft consensus layer, the consistent hash ring, the chaos engine, anomaly detection, incident
correlation, and the in-process benchmark harness. There is no external database or message broker —
all engines communicate through an in-memory event bus, and the persistent-machine CSIs (Postgres,
Redis, Ollama) are optional future integrations. This makes Atlas trivially runnable on a single
machine while still exercising real distributed-systems ideas.

---

## Features

| Category | Capabilities |
|---|---|
| **Graph Engine** | BFS, DFS, Dijkstra (shortest path), A*, topological sort, cycle detection, connected components, Tarjan SCCs, articulation points, bridges — directed & undirected |
| **Data Structures** | Dynamic array, singly/doubly linked list, stack, queue, circular queue, binary min-heap, priority queue, chaining hash map, BST, AVL tree, trie, LRU cache |
| **Scheduler** | Resource-aware workload scheduling with transparent score breakdown (CPU/memory/GPU/network/health/affinity + utilization penalty + priority bonus) |
| **Consistent Hashing** | Hash ring with virtual nodes for shard placement |
| **Distributed KV Store** | Replicated key–value service over the event bus |
| **Raft Consensus** | Leader election, log replication, heartbeats, term management, snapshots |
| **Job Queue** | Priority-based job processing with retries |
| **Topology & Services** | Node registry, health scores, dependency graph, services view |
| **Chaos Engineering** | kill_node, network_partition, latency, cpu_stress, memory_pressure — automatic revert, blast-radius impact analysis |
| **Anomaly Detection** | Rolling-window z-score detection + absolute threshold rules (default cpu > 90, memory_usage > 85) |
| **Incident Management** | Event correlation → incidents, coalescing, cluster-wide correlation, lifecycle (open → investigating → resolved) |
| **AI Advisory** | Rule-based archetype analysis (deliberate failure injection, dependency cascade, single-component failure, cluster-wide pressure, capacity bottleneck, latency degradation) — read-only, always requires approval |
| **Observability** | Prometheus text + JSON metrics endpoints, event counters, runtime collector, structured logging |
| **Benchmark Harness** | In-process timing of all 16 algorithm workloads with recorded runtime / memory / throughput |
| **CLI + Web Dashboard** | Operational CLI (`atlas-cli`) and a Next.js dark-theme dashboard |

## Architecture

```
                ┌───────────────────────────────────────────────┐
                │            Atlas CLI (atlas-cli)              │
                └───────────────────────┬───────────────────────┘
                                        │ HTTP (default :8080)
┌───────────────────────────────────────▼───────────────────────────────────┐
│                        Atlas Server (REST API — chi)                       │
│  /api/v1  health · nodes · topology · services · jobs · cluster · raft     │
│            incidents · anomalies · chaos · benchmarks · metrics            │
└───┬──────────┬──────────┬──────────┬──────────┬──────────────┬────────────┘
    ▼          ▼          ▼          ▼          ▼              ▼
┌─────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌───────────┐ ┌──────────────┐
│Graph &  │ │Sched-  │ │ Job    │ │ Raft   │ │  Chaos    │ │ Anomaly +    │
│Topology │ │uler    │ │ Queue  │ │Consensus│ │  Engine   │ │ Incident Mgr │
└─────────┴───────────┴────────┘ └────────┘ └───────────┘ └──────┬───────┘
        ▲          ▲         ▲         ▲                        │
        └──────────┴─────────┴─────────┴────────────────────────┘
                       Event Bus (publish/subscribe)
```

**Heartbeat through-line:** a chaos experiment → emits an event → anomaly / incident manager
correlates it into an incident → `incidents analyze` runs the AI advisor → everything is visible in
the web dashboard and observable via `/metrics`.

## Repository Layout

```
├── cmd/
│   ├── atlas-server/     # HTTP API server (composition root)
│   └── atlas-cli/        # Operational CLI
├── internal/
│   ├── algorithms/       # From-scratch DSA + graph + hashing + cache
│   ├── ai/               # Advisory AI analysis (rule-based archetypes)
│   ├── anomaly/          # Z-score + absolute-rule anomaly detector
│   ├── api/              # chi router + HTTP handlers
│   ├── benchmarks/       # In-process benchmark runner
│   ├── chaos/            # Failure-injection engine
│   ├── config/           # Config + environment overrides
│   ├── consensus/raft/   # Raft implementation
│   ├── event/            # In-memory event bus
│   ├── incident/         # Incident correlation manager
│   ├── jobqueue/         # Job queue
│   ├── kvstore/          # Distributed KV store service
│   ├── logger/           # slog setup
│   ├── metrics/          # Prometheus/JSON metrics registry
│   ├── middleware/       # Request ID, logger, recover, CORS
│   ├── models/           # Shared models
│   ├── scheduler/        # Workload scheduler
│   └── topology/         # Infrastructure graph engine
├── pkg/errors/           # Error helpers
├── web/                  # Next.js dashboard
├── docs/                 # ADRs, architecture, algorithms, PROGRESS
├── tests/                # Integration helpers
├── build/                # Prebuilt Windows binaries
└── Makefile              # dev/build/test/lint/benchmark targets
```

## Prerequisites

- **Go 1.27+** (required)
- **Node.js 20+** (only for the web dashboard)
- **make** optional — every target has a plain `go ...` equivalent

No database, broker, or AI service is required to run Atlas.

## Getting Started

### 1. Start the server

```bash
go run ./cmd/atlas-server        # or: make dev
```

Defaults to `0.0.0.0:8080`. Confirm it is healthy:

```bash
curl http://localhost:8080/api/v1/health
# {"status":"healthy","time":...}
```

### 2. Use the CLI

```bash
go run ./cmd/atlas-cli status     # health + info
go run ./cmd/atlas-cli nodes add --name db-1 --cpu 8 --memory 16
go run ./cmd/atlas-cli topology
```

### 3. Start the web dashboard

```bash
cd web
npm install
npm run dev       # http://localhost:3000
```

The dashboard polls the API at `http://localhost:8080/api/v1` (override with
`NEXT_PUBLIC_ATLAS_API_URL`).

### 4. Build binaries

```bash
make build                       # produces bin/atlas-server, bin/atlas-cli
go build -o build/atlas-server.exe ./cmd/atlas-server   # Windows, equivalent
go build -o build/atlas-cli.exe   ./cmd/atlas-cli
```

## Configuration

Configuration loads from an optional JSON file (`-config path.json`) plus environment overrides.
See `internal/config/config.go` for the full schema and defaults.

| Env var | Default | Purpose |
|---|---|---|
| `ATLAS_HOST` / `ATLAS_PORT` | `0.0.0.0` / `8080` | Server listen address |
| `ATLAS_LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `ATLAS_RAFT_NODE_ID` | `node-1` | Raft node identity |
| `ATLAS_AI_ENABLED` | `false` | Reserved for future LLM integration (advisory is rule-based today) |
| `ATLAS_DB_*` | — | Reserved for future Postgres integration |
| `ATLAS_REDIS_ADDR` | — | Reserved for future Redis integration |

The CLI also honors `ATLAS_API_URL` (default `http://localhost:8080`).

## End-to-End Walkthrough

1. **Register infrastructure**

   ```bash
   atlas nodes add --name app-1 --type server --cpu 8 --memory 16
   atlas nodes add --name db-1  --type database --cpu 4 --memory 8
   atlas topology
   ```

2. **Schedule work**

   ```bash
   atlas jobs create --type SCAN_NETWORK --priority 5 --payload '{"target":"db-1"}'
   atlas jobs
   ```

3. **Inject a failure and watch it become an incident**

   ```bash
   atlas chaos run --type kill_node --target db-1 --duration 60s
   atlas incidents --open          # incident auto-opened by correlation
   ```

   Anomaly detection is triggerable too:

   ```bash
   atlas anomalies rule --metric cpu --high 90
   atlas anomalies ingest --node app-1 --metric cpu --value 97
   ```

4. **Get advisory AI analysis**

   ```bash
   atlas incidents analyze <id>    # archetype + confidence, requires approval
   atlas incidents resolve <id> --reason "recovered after failover"
   ```

5. **Benchmark the algorithm library**

   ```bash
   atlas benchmarks names          # 16 workloads
   atlas benchmarks run hashtable_put_get
   atlas benchmarks run all        # record runtime / memory / throughput for each
   ```

6. **Observe**

   ```bash
   atlas metrics
   curl http://localhost:8080/metrics        # Prometheus text format
   ```

## CLI Reference

```
atlas <command> [commands] [--options]     env: ATLAS_API_URL

  status                       Show system health and info
  nodes add --name <n> [--id] [--type] [--health] [--cpu] [--memory]
  nodes rm <id>                Delete a node
  topology / services / cluster / raft
  jobs create --type <T> [--priority] [--payload]
  chaos run --type <kill_node|network_partition|latency|cpu_stress|memory_pressure>
            --target <node> [--duration 60s]
  chaos stop <id>
  incidents [--open] · incidents investigate <id> · resolve <id> --reason "..."
  anomalies ingest --node <id> --metric <m> --value <v> · anomalies rule --metric <m> [--high][--low]
  benchmarks run <name|all> · benchmarks names
  metrics · help · version
```

## API Reference

All endpoints are under `/api/v1` unless noted.

| Method | Path | Description |
|---|---|---|
| GET | `/health`, `/ready`, `/info` | Liveness, readiness, system info |
| GET/POST | `/nodes` | List or create nodes |
| GET/DELETE | `/nodes/{id}` | Get or delete a node |
| GET | `/topology` | Full infrastructure graph |
| GET | `/services` | Services view |
| GET/POST | `/jobs` | List or create jobs (payload is a raw JSON string) |
| GET | `/jobs/{id}` | Job details |
| GET | `/cluster`, `/raft` | Cluster and consensus state |
| GET | `/incidents` · `?open=true` | List all (or open) incidents |
| GET | `/incidents/{id}` | Incident details with correlated signals |
| POST | `/incidents/{id}/investigate` · `/resolve` · `/analyze` | Lifecycle + advisory analysis |
| GET | `/anomalies` | Detected anomalies |
| POST | `/anomalies/ingest` | Ingest a metric sample (202 when anomaly fires) |
| POST | `/anomalies/rules` | Add an absolute threshold rule |
| GET/POST | `/chaos`, GET `/chaos/active`, POST `/chaos/{id}/stop` | Chaos experiments |
| GET | `/benchmarks` · `/benchmarks/name` | Results / available workloads |
| POST | `/benchmarks/{algorithm}` · `/benchmarks/all` | Run one or all benchmarks |
| GET | `/metrics` | JSON metrics |
| GET | `/metrics` (top-level) | Prometheus text format |

## Testing

```bash
make check          # go vet + go test ./...
make test           # go test ./...
make test-race      # race detector (runs in CI on Linux)
make benchmark      # standard Go benchmarks
```

The test suite covers every algorithm package, the scheduler, raft (election + replication +
snapshots), job queue, kv store, topology, chaos, anomaly detection, incident correlation + AI
analysis, metrics, benchmarks, and full API lifecycle tests. `docs/PROGRESS.md` tracks the
milestone-by-milestone status.

> **Windows note:** `go test -race` requires gcc/cgo and is therefore run in CI instead of locally.

## Web Dashboard

`web/` is a Next.js (React 19, Tailwind 4) static client showing:

- **Components KPI** — registered nodes and resources
- **Topology** — infrastructure graph from `/topology`
- **Open Incidents** — correlated incidents with AI analysis
- **Anomalies** — recent detections
- **Running Chaos** — active experiments
- **Jobs** and **Nodes** tables

```bash
cd web && npm install && npm run build && npm run start   # production
cd web && npm run dev                                     # development
cd web && npm run lint                                    # eslint
```

## Design Decisions

Architecture and safety decisions are recorded as ADRs in `docs/decisions/`:

- ADR-001 — Go as the implementation language
- ADR-002 — Graph representation (adjacency)
- ADR-003 — Raft design (non-blocking elections, combined lock for leader snapshot broadcast)
- ADR-004 — Consistent hashing
- ADR-005 — Scheduler scoring (headroom-aware resource scoring + deterministic tie-break)
- ADR-006 — Event architecture (in-memory bus drives every engine)
- ADR-007 — **AI safety boundary** (read-only advisory, every recommendation requires approval, no remediation path)
- ADR-008 — Incident correlation & advisory analysis (windowed coalescing, cluster-wide threshold supersedes singletons)

## Known Limitations

- Persistent adapters (Postgres, Redis) and external LLM (Ollama) are **reserved but not wired**; AI analysis is rule-based, using topology snapshots.
- Benchmark numbers are measured on the local machine; treat them as relative indicators.
- Windows Application Control can intermittently block freshly compiled binaries; running via `go run` or rebuilding usually clears it.
- The event bus and all engines are in-memory — a restart clears state (intended for a laboratory sandbox; persistence is a follow-up milestone).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE).
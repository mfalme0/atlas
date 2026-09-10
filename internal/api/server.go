package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/atlas-engine/atlas/internal/ai"
	"github.com/atlas-engine/atlas/internal/anomaly"
	"github.com/atlas-engine/atlas/internal/benchmarks"
	"github.com/atlas-engine/atlas/internal/chaos"
	"github.com/atlas-engine/atlas/internal/config"
	"github.com/atlas-engine/atlas/internal/incident"
	"github.com/atlas-engine/atlas/internal/jobqueue"
	"github.com/atlas-engine/atlas/internal/metrics"
	"github.com/atlas-engine/atlas/internal/middleware"
	"github.com/atlas-engine/atlas/internal/models"
	"github.com/atlas-engine/atlas/internal/topology"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Dependencies wires external engines into the API layer.
type Dependencies struct {
	Topology *topology.Engine
	Queue    *jobqueue.Queue
	Chaos    *chaos.Engine
	Anomaly  *anomaly.Detector
	Incident *incident.Manager
	AI       *ai.Advisor
	Metrics  *metrics.Registry
	Bench    *benchmarks.Runner
}

type Server struct {
	cfg    *config.Config
	log    *slog.Logger
	deps   Dependencies
	router *chi.Mux
	start  time.Time
}

func New(cfg *config.Config, log *slog.Logger) *Server {
	return NewWithDeps(cfg, log, Dependencies{})
}

func NewWithDeps(cfg *config.Config, log *slog.Logger, deps Dependencies) *Server {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{
		cfg:   cfg,
		log:   log,
		deps:  deps,
		start: time.Now(),
	}
	s.router = s.buildRouter()
	return s
}

func (s *Server) Router() *chi.Mux {
	return s.router
}

func (s *Server) buildRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(s.log))
	middlewareFunc := func(next http.Handler) http.Handler {
		return middleware.Recover(s.log)(next)
	}
	r.Use(middlewareFunc)
	r.Use(middleware.CORS)
	r.Use(chimw.Heartbeat("/healthz"))
	r.Use(chimw.Timeout(30 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/ready", s.handleReady)
		r.Get("/info", s.handleInfo)
		r.Get("/metrics", s.handleMetricsJSON)

		r.Route("/nodes", func(r chi.Router) {
			r.Get("/", s.handleListNodes)
			r.Post("/", s.handleCreateNode)
			r.Get("/{id}", s.handleGetNode)
			r.Delete("/{id}", s.handleDeleteNode)
		})

		r.Route("/topology", func(r chi.Router) {
			r.Get("/", s.handleGetTopology)
		})

		r.Route("/services", func(r chi.Router) {
			r.Get("/", s.handleListServices)
		})

		r.Route("/jobs", func(r chi.Router) {
			r.Get("/", s.handleListJobs)
			r.Post("/", s.handleCreateJob)
			r.Get("/{id}", s.handleGetJob)
		})

		r.Route("/cluster", func(r chi.Router) {
			r.Get("/", s.handleClusterStatus)
		})

		r.Route("/raft", func(r chi.Router) {
			r.Get("/", s.handleRaftStatus)
		})

		r.Route("/incidents", func(r chi.Router) {
			r.Get("/", s.handleListIncidents)
			r.Get("/{id}", s.handleGetIncident)
			r.Post("/{id}/investigate", s.handleInvestigate)
			r.Post("/{id}/resolve", s.handleResolve)
			r.Post("/{id}/analyze", s.handleAnalyze)
		})
		r.Route("/anomalies", func(r chi.Router) {
			r.Get("/", s.handleListAnomalies)
			r.Post("/ingest", s.handleIngestSample)
			r.Post("/rules", s.handleAddRule)
		})
		r.Route("/chaos", func(r chi.Router) {
			r.Get("/", s.handleListChaos)
			r.Post("/", s.handleChaosExperiment)
			r.Get("/active", s.handleActiveChaos)
			r.Post("/{id}/stop", s.handleStopChaos)
		})

		r.Route("/benchmarks", func(r chi.Router) {
			r.Get("/", s.handleListBenchmarks)
			r.Get("/name", s.handleBenchmarkNames)
			r.Post("/all", s.handleRunAllBenchmarks)
			r.Post("/{algorithm}", s.handleRunBenchmark)
		})
	})

	r.Get("/metrics", s.handleMetricsPrometheus)

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ready",
		"version": "0.1.0",
	})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":     "atlas",
		"version":  "0.1.0",
		"uptime":   time.Since(s.start).String(),
		"go":       runtime.Version(),
		"cpus":     runtime.NumCPU(),
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"alloc":      m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":        m.Sys,
			"gc_cycles":  m.NumGC,
		},
	})
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	if s.deps.Topology == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": []interface{}{}, "total": 0})
		return
	}
	nodes := s.deps.Topology.Nodes()
	writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": nodes, "total": len(nodes)})
}

func (s *Server) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	if s.deps.Topology == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "topology engine not wired"})
		return
	}
	var node models.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid node payload"})
		return
	}
	if node.ID == "" || node.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "node requires id and name"})
		return
	}
	s.deps.Topology.AddNode(&node)
	writeJSON(w, http.StatusCreated, node)
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	if s.deps.Topology == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "node not found"})
		return
	}
	id := chi.URLParam(r, "id")
	node, ok := s.deps.Topology.GetNode(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "node not found"})
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	if s.deps.Topology == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "node not found"})
		return
	}
	id := chi.URLParam(r, "id")
	s.deps.Topology.RemoveNode(id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": id})
}

func (s *Server) handleGetTopology(w http.ResponseWriter, r *http.Request) {
	if s.deps.Topology == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		})
		return
	}
	snap := s.deps.Topology.Snapshot()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"nodes":        snap.Nodes,
		"edges":        snap.Edges,
		"health_score": snap.HealthScore,
		"bottlenecks":  s.deps.Topology.Bottlenecks(),
		"cycles":       s.deps.Topology.Cycles(),
		"timestamp":    snap.Timestamp,
	})
}

func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"services": []interface{}{},
		"total":    0,
	})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queue == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"jobs": []interface{}{}, "total": 0})
		return
	}
	jobs := s.deps.Queue.Jobs()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"total": len(jobs),
		"stats": s.deps.Queue.Stats(),
	})
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queue == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "job queue not wired"})
		return
	}
	var job jobqueue.Job
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "invalid job payload"})
		return
	}
	if job.ID == "" {
		job.ID = fmt.Sprintf("job-%d", time.Now().UnixNano())
	}
	if job.Name == "" {
		job.Name = job.Type
	}
	if err := s.deps.Queue.Submit(&job); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, jobqueue.ErrDuplicate) {
			status = http.StatusConflict
		} else if errors.Is(err, jobqueue.ErrInvalid) {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if s.deps.Queue == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "job not found"})
		return
	}
	id := chi.URLParam(r, "id")
	job, err := s.deps.Queue.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"nodes": []interface{}{},
		"state": "single_node",
	})
}

func (s *Server) handleRaftStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"state":    "follower",
		"term":     0,
		"leader":   "",
		"log_index": 0,
	})
}

type ingestSampleRequest struct {
	NodeID string  `json:"node_id"`
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

func (s *Server) handleListIncidents(w http.ResponseWriter, r *http.Request) {
	if s.deps.Incident == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"incidents": []interface{}{}, "total": 0})
		return
	}
	openOnly := r.URL.Query().Get("open") == "true"
	incidents := s.deps.Incident.List(openOnly)
	writeJSON(w, http.StatusOK, map[string]interface{}{"incidents": incidents, "total": len(incidents)})
}

func (s *Server) handleGetIncident(w http.ResponseWriter, r *http.Request) {
	if s.deps.Incident == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "incident manager not wired"})
		return
	}
	inc := s.deps.Incident.Get(chi.URLParam(r, "id"))
	if inc == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "incident not found"})
		return
	}
	writeJSON(w, http.StatusOK, inc)
}

func (s *Server) handleInvestigate(w http.ResponseWriter, r *http.Request) {
	if s.deps.Incident == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "incident manager not wired"})
		return
	}
	if !s.deps.Incident.Investigate(chi.URLParam(r, "id")) {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "incident not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "investigating"})
}

func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	if s.deps.Incident == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "incident manager not wired"})
		return
	}
	var body struct {
		Resolution string `json:"resolution"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "resolution must be a JSON object"})
		return
	}
	if !s.deps.Incident.Resolve(chi.URLParam(r, "id"), body.Resolution) {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "incident not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "resolved"})
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if s.deps.Incident == nil || s.deps.AI == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "analysis not wired"})
		return
	}
	id := chi.URLParam(r, "id")
	inc := s.deps.Incident.Get(id)
	if inc == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "incident not found"})
		return
	}
	signals := s.deps.Incident.Signals(id)
	analysis := s.deps.AI.Analyze(inc, signals, s.deps.Topology)
	s.deps.Incident.SetAnalysis(id, summarizeAnalysis(analysis), analysis.RootCause)
	writeJSON(w, http.StatusOK, analysis)
}

func summarizeAnalysis(a *ai.Analysis) string {
	recs := make([]string, 0, len(a.Recommendations))
	for _, r := range a.Recommendations {
		recs = append(recs, r.Action)
	}
	return fmt.Sprintf("%s (confidence %.2f) - %s | advisories: %s",
		a.Archetype, a.Confidence, a.RootCause, strings.Join(recs, "; "))
}

func (s *Server) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	if s.deps.Anomaly == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"anomalies": []interface{}{}, "total": 0})
		return
	}
	history := s.deps.Anomaly.History()
	writeJSON(w, http.StatusOK, map[string]interface{}{"anomalies": history, "total": len(history)})
}

func (s *Server) handleIngestSample(w http.ResponseWriter, r *http.Request) {
	if s.deps.Anomaly == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "anomaly engine not wired"})
		return
	}
	var req ingestSampleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NodeID == "" || req.Metric == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "node_id and metric are required"})
		return
	}
	detected := s.deps.Anomaly.Record(req.NodeID, req.Metric, req.Value)
	status := http.StatusOK
	if detected != nil {
		status = http.StatusAccepted
	}
	writeJSON(w, status, map[string]interface{}{
		"ingested": true,
		"anomaly":  detected,
	})
}

func (s *Server) handleAddRule(w http.ResponseWriter, r *http.Request) {
	if s.deps.Anomaly == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "anomaly engine not wired"})
		return
	}
	var rule anomaly.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil || rule.Metric == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "metric is required"})
		return
	}
	s.deps.Anomaly.AddRule(rule)
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) handleChaosExperiment(w http.ResponseWriter, r *http.Request) {
	if s.deps.Chaos == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "chaos engine not wired"})
		return
	}
	var exp chaos.Experiment
	if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{"error": "invalid experiment payload"})
		return
	}
	if exp.ID == "" || exp.Type == "" || exp.Target == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "experiment requires id, type, and target"})
		return
	}
	if err := s.deps.Chaos.Run(context.Background(), &exp); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, exp)
}

func (s *Server) handleListChaos(w http.ResponseWriter, r *http.Request) {
	if s.deps.Chaos == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"experiments": []interface{}{}, "total": 0})
		return
	}
	results := s.deps.Chaos.Results()
	writeJSON(w, http.StatusOK, map[string]interface{}{"experiments": results, "total": len(results)})
}

func (s *Server) handleActiveChaos(w http.ResponseWriter, r *http.Request) {
	if s.deps.Chaos == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"experiments": []interface{}{}, "total": 0})
		return
	}
	active := s.deps.Chaos.Active()
	writeJSON(w, http.StatusOK, map[string]interface{}{"experiments": active, "total": len(active)})
}

func (s *Server) handleStopChaos(w http.ResponseWriter, r *http.Request) {
	if s.deps.Chaos == nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "chaos engine not wired"})
		return
	}
	id := chi.URLParam(r, "id")
	s.deps.Chaos.Stop(id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"stopped": id})
}

func (s *Server) handleListBenchmarks(w http.ResponseWriter, r *http.Request) {
	if s.deps.Bench == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"benchmarks": []interface{}{}, "total": 0})
		return
	}
	results := s.deps.Bench.List()
	writeJSON(w, http.StatusOK, map[string]interface{}{"benchmarks": results, "total": len(results)})
}

func (s *Server) handleBenchmarkNames(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"algorithms": benchmarks.Names(), "total": len(benchmarks.Names())})
}

func (s *Server) handleRunBenchmark(w http.ResponseWriter, r *http.Request) {
	if s.deps.Bench == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "benchmark runner not wired"})
		return
	}
	name := chi.URLParam(r, "algorithm")
	res, err := s.deps.Bench.Run(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleRunAllBenchmarks(w http.ResponseWriter, r *http.Request) {
	if s.deps.Bench == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]interface{}{"error": "benchmark runner not wired"})
		return
	}
	results := s.deps.Bench.RunAll()
	writeJSON(w, http.StatusOK, map[string]interface{}{"benchmarks": results, "total": len(results)})
}

func (s *Server) handleMetricsPrometheus(w http.ResponseWriter, r *http.Request) {
	if s.deps.Metrics == nil {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	s.deps.Metrics.RenderPrometheus(w)
}

func (s *Server) handleMetricsJSON(w http.ResponseWriter, r *http.Request) {
	if s.deps.Metrics == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"counters": map[string]int64{}, "gauges": map[string]float64{}})
		return
	}
	reg := s.deps.Metrics
	counters := make(map[string]int64)
	for _, name := range reg.CounterNames() {
		counters[name] = reg.Counter(name)
	}
	gauges := make(map[string]float64)
	for _, name := range reg.GaugeNames() {
		gauges[name] = reg.GaugeValue(name)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"counters": counters,
		"gauges":   gauges,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atlas-engine/atlas/internal/ai"
	"github.com/atlas-engine/atlas/internal/anomaly"
	"github.com/atlas-engine/atlas/internal/benchmarks"
	"github.com/atlas-engine/atlas/internal/chaos"
	"github.com/atlas-engine/atlas/internal/config"
	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/incident"
	"github.com/atlas-engine/atlas/internal/jobqueue"
	"github.com/atlas-engine/atlas/internal/metrics"
	"github.com/atlas-engine/atlas/internal/models"
	"github.com/atlas-engine/atlas/internal/topology"
)

func testConfig() *config.Config {
	cfg, _ := config.Load("")
	return cfg
}

func testServer() *Server {
	bus := event.NewBus(16)
	topo := topology.NewEngine(bus)
	queue := jobqueue.NewQueue(jobqueue.Options{})
	chaosEngine := chaos.NewEngine(topo, bus, nil)
	anomalyEngine := anomaly.NewDetector(anomaly.Options{Bus: bus})
	incidentManager := incident.NewManager(incident.Options{Bus: bus})
	incidentManager.Start()
	advisor := ai.NewAdvisor(3)
	metricsReg := metrics.NewRegistry()
	metrics.NewBusRecorder(bus, metricsReg)
	return NewWithDeps(testConfig(), nil, Dependencies{
		Topology: topo,
		Queue:    queue,
		Chaos:    chaosEngine,
		Anomaly:  anomalyEngine,
		Incident: incidentManager,
		AI:       advisor,
		Metrics:  metricsReg,
		Bench:    benchmarks.NewRunner(),
	})
}

func getRequest(t *testing.T, s *Server, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	return rec
}

func TestTopologyNodesCRUD(t *testing.T) {
	s := testServer()

	rec := getRequest(t, s, http.MethodPost, "/api/v1/nodes", &models.Node{
		ID:   "db-1",
		Name: "postgres-primary",
		Type: models.NodeTypeDatabase,
		Health: 1.0,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create node got %d", rec.Code)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/nodes/db-1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get node got %d", rec.Code)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/nodes", nil)
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("expected 1 node got %d", list.Total)
	}

	rec = getRequest(t, s, http.MethodDelete, "/api/v1/nodes/db-1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete node got %d", rec.Code)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/nodes/db-1", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete got %d", rec.Code)
	}
}

func TestTopologySnapshot(t *testing.T) {
	s := testServer()

	topo := s.deps.Topology
	topo.AddNode(&models.Node{ID: "web", Name: "web", Type: models.NodeTypeApp, Health: 1.0})
	topo.AddNode(&models.Node{ID: "db", Name: "db", Type: models.NodeTypeDatabase, Health: 0.5})
	topo.AddEdge(&models.Edge{ID: "e1", Source: "web", Target: "db", Type: models.EdgeDependsOn, Weight: 1})

	rec := getRequest(t, s, http.MethodGet, "/api/v1/topology", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("topology got %d", rec.Code)
	}
	var resp struct {
		Nodes       []*models.Node `json:"nodes"`
		Edges       []*models.Edge `json:"edges"`
		HealthScore float64        `json:"health_score"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode topology: %v", err)
	}
	if len(resp.Nodes) != 2 || len(resp.Edges) != 1 {
		t.Fatalf("expected 2 nodes/1 edge got %d/%d", len(resp.Nodes), len(resp.Edges))
	}
	if resp.HealthScore != 0.75 {
		t.Fatalf("expected health 0.75 got %v", resp.HealthScore)
	}
}

func TestJobsCRUD(t *testing.T) {
	s := testServer()

	rec := getRequest(t, s, http.MethodPost, "/api/v1/jobs", &jobqueue.Job{
		ID:   "job-1",
		Name: "backup",
		Type: "backup",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create job got %d", rec.Code)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/jobs/job-1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get job got %d", rec.Code)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/jobs", nil)
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode jobs list: %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("expected 1 job got %d", list.Total)
	}

	// duplicate should 409
	rec = getRequest(t, s, http.MethodPost, "/api/v1/jobs", &jobqueue.Job{
		ID:   "job-1",
		Name: "backup",
		Type: "backup",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate job expected 409 got %d", rec.Code)
	}
}

func TestUnwiredEngines(t *testing.T) {
	s := New(testConfig(), nil)

	rec := getRequest(t, s, http.MethodPost, "/api/v1/jobs", &jobqueue.Job{ID: "x", Name: "x"})
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 for unwired queue got %d", rec.Code)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/nodes", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 empty list got %d", rec.Code)
	}
}

func TestHealthReadyInfo(t *testing.T) {
	s := testServer()
	for _, path := range []string{"/api/v1/health", "/api/v1/ready", "/api/v1/info", "/healthz"} {
		rec := getRequest(t, s, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s got %d", path, rec.Code)
		}
	}
}

func TestAnomalyIngestAndRules(t *testing.T) {
	s := testServer()

	rec := getRequest(t, s, http.MethodPost, "/api/v1/anomalies/rules", &anomaly.Rule{Metric: "cpu", CriticalHigh: 90})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add rule got %d: %s", rec.Code, rec.Body.String())
	}

	for i := 0; i < 20; i++ {
		rec = getRequest(t, s, http.MethodPost, "/api/v1/anomalies/ingest", &ingestSampleRequest{NodeID: "web-1", Metric: "cpu", Value: 30})
		if rec.Code != http.StatusOK {
			t.Fatalf("ingest baseline got %d", rec.Code)
		}
	}

	rec = getRequest(t, s, http.MethodPost, "/api/v1/anomalies/ingest", &ingestSampleRequest{NodeID: "web-1", Metric: "cpu", Value: 98})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 on detection, got %d", rec.Code)
	}
	var body struct {
		Anomaly *anomaly.Anomaly `json:"anomaly"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode detection: %v", err)
	}
	if body.Anomaly == nil || body.Anomaly.NodeID != "web-1" || body.Anomaly.Severity != anomaly.SeverityCritical {
		t.Fatalf("unexpected anomaly: %+v", body.Anomaly)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/anomalies", nil)
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("expected 1 anomaly in history, got %d", list.Total)
	}

	rec = getRequest(t, s, http.MethodPost, "/api/v1/anomalies/ingest", &ingestSampleRequest{Metric: "cpu"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing node_id, got %d", rec.Code)
	}
}

func TestMetricsEndpoints(t *testing.T) {
	s := testServer()
	s.deps.Metrics.Inc("atlas_events_total", 12)
	s.deps.Metrics.Gauge("atlas_health_score", 0.87)

	rec := getRequest(t, s, http.MethodGet, "/api/v1/metrics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("json metrics got %d", rec.Code)
	}
	var body struct {
		Counters map[string]int64  `json:"counters"`
		Gauges   map[string]float64 `json:"gauges"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Counters["atlas_events_total"] != 12 {
		t.Fatalf("expected counter 12, got %d", body.Counters["atlas_events_total"])
	}

	rec = getRequest(t, s, http.MethodGet, "/metrics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("prometheus metrics got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !bytes.Contains([]byte(ct), []byte("text/plain")) {
		t.Fatalf("expected text/plain content type, got %q", ct)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("atlas_events_total 12")) {
		t.Fatalf("missing prometheus value in:\n%s", rec.Body.String())
	}
}

func TestBenchmarksEndpoints(t *testing.T) {
	s := testServer()

	rec := getRequest(t, s, http.MethodGet, "/api/v1/benchmarks/name", nil)
	var names struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &names); err != nil {
		t.Fatalf("decode names: %v", err)
	}
	if names.Total < 10 {
		t.Fatalf("expected >= 10 benchmark names, got %d", names.Total)
	}

	rec = getRequest(t, s, http.MethodPost, "/api/v1/benchmarks/array_append", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("run benchmark got %d: %s", rec.Code, rec.Body.String())
	}
	var res models.BenchmarkResult
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if res.Algorithm != "array_append" || res.Runtime <= 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/benchmarks", nil)
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("expected 1 recorded benchmark, got %d", list.Total)
	}

	rec = getRequest(t, s, http.MethodPost, "/api/v1/benchmarks/does_not_exist", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown benchmark, got %d", rec.Code)
	}
}

func TestIncidentLifecycleAndAnalysis(t *testing.T) {
	s := testServer()

	s.deps.Incident.RecordSignal(incident.Signal{Kind: "node_failed", NodeID: "db-1", Severity: incident.SeverityCritical})

	s.deps.Topology.AddNode(&models.Node{ID: "db-1", Name: "db-1", Type: models.NodeTypeDatabase, State: models.NodeStateFailed, Health: 0})

	rec := getRequest(t, s, http.MethodGet, "/api/v1/incidents", nil)
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode incidents: %v", err)
	}
	if list.Total != 1 {
		t.Fatalf("expected 1 incident, got %d", list.Total)
	}

	id := s.deps.Incident.List(false)[0].ID

	rec = getRequest(t, s, http.MethodPost, "/api/v1/incidents/"+id+"/investigate", map[string]interface{}{})
	if rec.Code != http.StatusOK {
		t.Fatalf("investigate got %d", rec.Code)
	}

	rec = getRequest(t, s, http.MethodPost, "/api/v1/incidents/"+id+"/analyze", map[string]interface{}{})
	if rec.Code != http.StatusOK {
		t.Fatalf("analyze got %d: %s", rec.Code, rec.Body.String())
	}
	var analysis ai.Analysis
	if err := json.Unmarshal(rec.Body.Bytes(), &analysis); err != nil {
		t.Fatalf("decode analysis: %v", err)
	}
	if analysis.Archetype != ai.ArchetypeSingleFailure {
		t.Fatalf("expected single failure archetype, got %q", analysis.Archetype)
	}
	for _, r := range analysis.Recommendations {
		if !r.RequiresApproval {
			t.Fatalf("recommendations must require approval: %+v", r)
		}
	}

	inc := s.deps.Incident.Get(id)
	if inc.AIAnalysis == "" {
		t.Fatalf("expected analysis text stored on incident")
	}

	rec = getRequest(t, s, http.MethodPost, "/api/v1/incidents/"+id+"/resolve", map[string]interface{}{"resolution": "restarted db-1 after fsck"})
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve got %d", rec.Code)
	}
	inc = s.deps.Incident.Get(id)
	if inc.State != incident.StateResolved || inc.Resolution != "restarted db-1 after fsck" {
		t.Fatalf("resolve not applied: %+v", inc)
	}
}

func TestChaosExperimentLifecycle(t *testing.T) {
	s := testServer()
	s.deps.Topology.AddNode(&models.Node{ID: "db-1", Name: "db-1", Type: models.NodeTypeDatabase, Health: 1.0})

	rec := getRequest(t, s, http.MethodPost, "/api/v1/chaos", &chaos.Experiment{
		ID:       "chaos-1",
		Type:     chaos.TypeKillNode,
		Target:   "db-1",
		Duration: 5 * time.Second,
	})
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start chaos got %d: %s", rec.Code, rec.Body.String())
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/chaos/active", nil)
	var active struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &active); err != nil {
		t.Fatalf("decode active: %v", err)
	}
	if active.Total != 1 {
		t.Fatalf("expected 1 active experiment, got %d", active.Total)
	}

	node, _ := s.deps.Topology.GetNode("db-1")
	if node.Health != 0 {
		t.Fatalf("expected node killed during experiment, health=%v", node.Health)
	}

	rec = getRequest(t, s, http.MethodPost, "/api/v1/chaos/chaos-1/stop", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stop chaos got %d", rec.Code)
	}

	node, _ = s.deps.Topology.GetNode("db-1")
	if node.Health != 1.0 {
		t.Fatalf("expected node healthy after stop, health=%v", node.Health)
	}

	rec = getRequest(t, s, http.MethodGet, "/api/v1/chaos", nil)
	var results struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	if results.Total != 1 {
		t.Fatalf("expected 1 recorded experiment, got %d", results.Total)
	}
}

func TestChaosRejectsBadPayload(t *testing.T) {
	s := testServer()

	rec := getRequest(t, s, http.MethodPost, "/api/v1/chaos", &chaos.Experiment{ID: "x"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing type/target, got %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chaos", bytes.NewBufferString("{not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for invalid JSON, got %d", rec.Code)
	}
}
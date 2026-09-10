// Package incident correlates signals — anomalies, node failures, chaos
// experiments, job failures — into incidents tracked through a lifecycle
// (open -> investigating -> resolved).
//
// Coalescing rules:
//   - Signals on the same node within the correlation window merge into the
//     node's open incident.
//   - When the same metric is anomalous on at least ClusterThreshold distinct
//     nodes within the window, a cluster-wide incident is opened instead.
//
// Incident creation emits an incident.created event on the event bus.
package incident

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
)

// Severity labels used across incidents.
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
)

// State labels of the incident lifecycle.
const (
	StateOpen          = "open"
	StateInvestigating = "investigating"
	StateResolved      = "resolved"
)

// Options configure a Manager.
type Options struct {
	Window          time.Duration // correlation window for coalescing
	ClusterThreshold int           // distinct nodes for a cluster-wide incident
	Bus             *event.Bus
	Logger          *slog.Logger
	Clock           func() time.Time
}

// Signal is a single observed symptom attached to an incident.
type Signal struct {
	Kind     string    `json:"kind"` // anomaly | node_failed | job_failed | chaos
	NodeID   string    `json:"node_id"`
	Metric   string    `json:"metric,omitempty"`
	Value    float64   `json:"value,omitempty"`
	Severity string    `json:"severity"`
	Detail   string    `json:"detail,omitempty"`
	Time     time.Time `json:"time"`
}

// View is the read model of an incident exposed through the API.
type View struct {
	ID              string       `json:"id"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Severity        string       `json:"severity"`
	State           string       `json:"state"`
	AffectedNodes   []string     `json:"affected_nodes"`
	AIAnalysis      string       `json:"ai_analysis,omitempty"`
	RootCause       string       `json:"root_cause,omitempty"`
	Resolution      string       `json:"resolution,omitempty"`
	Signals         []Signal     `json:"signals"`
	CreatedAt       time.Time    `json:"created_at"`
	ResolvedAt      *time.Time   `json:"resolved_at,omitempty"`
}

type record struct {
	view    *View
	metrics map[string]bool
	lastAt  time.Time
}

// Manager tracks incidents.
type Manager struct {
	mu       sync.Mutex
	opts     Options
	records  map[string]*record
	byNode   map[string]string // node -> latest incident id
	order    []string
	next     int
}

// NewManager creates an incident manager.
func NewManager(opts Options) *Manager {
	if opts.Window <= 0 {
		opts.Window = 60 * time.Second
	}
	if opts.ClusterThreshold <= 0 {
		opts.ClusterThreshold = 3
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return &Manager{
		opts:   opts,
		records: make(map[string]*record),
		byNode: make(map[string]string),
		order:  make([]string, 0),
	}
}

// Start subscribes the manager to the event bus.
func (m *Manager) Start() {
	if m.opts.Bus == nil {
		return
	}
	m.opts.Bus.Subscribe(event.AnomalyDetected, m.handleAnomaly)
	m.opts.Bus.Subscribe(event.NodeFailed, m.handleNodeFailed)
	m.opts.Bus.Subscribe(event.JobFailed, m.handleJobFailed)
	m.opts.Bus.Subscribe(event.ChaosExperimentStarted, m.handleChaosStarted)
}

// RecordSignal ingests a symptom directly (bus-independent path).
func (m *Manager) RecordSignal(s Signal) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recordSignalLocked(s)
}

func (m *Manager) handleAnomaly(e event.Event) {
	sev := severityOf(e.Data["severity"])
	m.RecordSignal(Signal{
		Kind:     "anomaly",
		NodeID:   fmt.Sprint(e.Data["node_id"]),
		Metric:   fmt.Sprint(e.Data["metric"]),
		Value:    floatOf(e.Data["value"]),
		Severity: sev,
		Time:     e.Timestamp,
	})
}

func (m *Manager) handleNodeFailed(e event.Event) {
	m.RecordSignal(Signal{
		Kind:     "node_failed",
		NodeID:   fmt.Sprint(e.Data["node_id"]),
		Severity: SeverityCritical,
		Time:     e.Timestamp,
	})
}

func (m *Manager) handleJobFailed(e event.Event) {
	node := fmt.Sprint(e.Data["node_id"])
	if node == "" {
		node = fmt.Sprint(e.Data["worker_id"])
	}
	m.RecordSignal(Signal{
		Kind:     "job_failed",
		NodeID:   node,
		Severity: SeverityWarning,
		Detail:   fmt.Sprint(e.Data["error"]),
		Time:     e.Timestamp,
	})
}

func (m *Manager) handleChaosStarted(e event.Event) {
	m.RecordSignal(Signal{
		Kind:     "chaos",
		NodeID:   fmt.Sprint(e.Data["experiment_id"]),
		Severity: SeverityWarning,
		Detail:   "chaos experiment running",
		Time:     e.Timestamp,
	})
}

func (m *Manager) recordSignalLocked(s Signal) {
	if s.NodeID == "" {
		return
	}
	now := m.opts.Clock()

	if id, ok := m.byNode[s.NodeID]; ok {
		rec := m.records[id]
		if rec != nil && rec.view.State != StateResolved && now.Sub(rec.lastAt) <= m.opts.Window {
			m.attachSignal(rec, s)
			return
		}
	}

	// Cluster-wide correlation: same metric anomalous on distinct nodes.
	if s.Kind == "anomaly" && s.Metric != "" {
		corr := m.correlatedFor(s.Metric, s.NodeID, now)
		if len(corr.nodes) >= m.opts.ClusterThreshold {
			m.openClusterLocked(corr, s)
			return
		}
	}

	m.openLocked(singletonTitle(s), s.Detail, s.Severity, []string{s.NodeID}, s)
}

type metricCorrelation struct {
	nodes map[string]bool
	ids   []string // incident ids absorbed into the cluster incident
}

func (m *Manager) correlatedFor(metric, current string, now time.Time) metricCorrelation {
	corr := metricCorrelation{nodes: make(map[string]bool)}
	for _, id := range m.order {
		rec := m.records[id]
		if rec == nil || rec.view.State == StateResolved || !rec.metrics[metric] {
			continue
		}
		if now.Sub(rec.lastAt) <= m.opts.Window {
			corr.ids = append(corr.ids, id)
			for _, n := range rec.view.AffectedNodes {
				corr.nodes[n] = true
			}
		}
	}
	// The arriving signal's node has not been attached yet; count it too.
	if current != "" {
		corr.nodes[current] = true
	}
	return corr
}

func (m *Manager) openClusterLocked(corr metricCorrelation, s Signal) {
	now := m.opts.Clock()
	affected := make([]string, 0, len(corr.nodes))
	for n := range corr.nodes {
		affected = append(affected, n)
	}
	// Absorb correlated single-node incidents into the cluster incident.
	for _, id := range corr.ids {
		rec := m.records[id]
		if rec == nil || rec.view.State == StateResolved {
			continue
		}
		rec.view.State = StateResolved
		rec.view.Resolution = "superseded by cluster-wide correlation"
		t := now
		rec.view.ResolvedAt = &t
		for _, n := range rec.view.AffectedNodes {
			delete(m.byNode, n)
		}
	}
	clusterID := m.openLocked(clusterTitle(s.Metric), clusterDescription(s.Metric, len(affected)), SeverityCritical, affected, s)
	for _, n := range affected {
		m.byNode[n] = clusterID
	}
}

func (m *Manager) attachSignal(rec *record, s Signal) {
	rec.metrics[s.Metric] = true
	rec.view.Signals = append(rec.view.Signals, s)
	rec.view.Description = mergeDescription(rec.view.Description, s)
	if s.Severity == SeverityCritical && rec.view.Severity != SeverityCritical {
		rec.view.Severity = SeverityCritical
	}
	rec.lastAt = m.opts.Clock()
}

func (m *Manager) openLocked(title, description, severity string, affected []string, s Signal) string {
	m.next++
	id := fmt.Sprintf("inc-%d", m.next)
	now := m.opts.Clock()
	v := &View{
		ID:          id,
		Title:       title,
		Description: description,
		Severity:    severity,
		State:       StateOpen,
		AffectedNodes: dedupe(affected),
		Signals:     []Signal{s},
		CreatedAt:   now,
	}
	rec := &record{view: v, metrics: map[string]bool{}, lastAt: now}
	if s.Metric != "" {
		rec.metrics[s.Metric] = true
	}
	m.records[id] = rec
	m.order = append(m.order, id)
	for _, n := range affected {
		if n != "" {
			m.byNode[n] = id
		}
	}
	m.opts.Logger.Warn("incident opened",
		slog.String("id", id),
		slog.String("severity", severity),
		slog.String("title", title))
	m.emitCreated(v, s)
	return id
}

func (m *Manager) emitCreated(v *View, s Signal) {
	if m.opts.Bus == nil {
		return
	}
	m.opts.Bus.Publish(event.NewEvent(event.IncidentCreated, "incident", map[string]interface{}{
		"incident_id": v.ID,
		"title":       v.Title,
		"severity":    v.Severity,
		"node_id":     s.NodeID,
		"metric":      s.Metric,
	}))
}

// Investigate marks an incident as under investigation.
func (m *Manager) Investigate(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[id]
	if rec == nil {
		return false
	}
	rec.view.State = StateInvestigating
	return true
}

// Resolve marks an incident resolved with an optional human-authored resolution.
func (m *Manager) Resolve(id, resolution string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[id]
	if rec == nil {
		return false
	}
	rec.view.State = StateResolved
	rec.view.Resolution = resolution
	now := m.opts.Clock()
	rec.view.ResolvedAt = &now
	return true
}

// SetAnalysis attaches AI analysis output to an incident (advisory only).
func (m *Manager) SetAnalysis(id, analysis, rootCause string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[id]
	if rec == nil {
		return false
	}
	rec.view.AIAnalysis = analysis
	if rootCause != "" {
		rec.view.RootCause = rootCause
	}
	return true
}

// List returns incidents, newest first, optionally only open ones.
func (m *Manager) List(openOnly bool) []*View {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*View, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		rec := m.records[m.order[i]]
		if rec == nil {
			continue
		}
		if openOnly && rec.view.State == StateResolved {
			continue
		}
		v := *rec.view
		v.Signals = append([]Signal(nil), rec.view.Signals...)
		v.AffectedNodes = append([]string(nil), rec.view.AffectedNodes...)
		out = append(out, &v)
	}
	return out
}

// Get returns a single incident view.
func (m *Manager) Get(id string) *View {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[id]
	if rec == nil {
		return nil
	}
	v := *rec.view
	v.Signals = append([]Signal(nil), rec.view.Signals...)
	v.AffectedNodes = append([]string(nil), rec.view.AffectedNodes...)
	return &v
}

// Signals returns the signals attached to an incident for analysis.
func (m *Manager) Signals(id string) []Signal {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec := m.records[id]
	if rec == nil {
		return nil
	}
	return append([]Signal(nil), rec.view.Signals...)
}

func singletonTitle(s Signal) string {
	switch s.Kind {
	case "node_failed":
		return "Node failed: " + s.NodeID
	case "chaos":
		return "Chaos experiment active"
	case "job_failed":
		return "Job failure on " + s.NodeID
	default:
		return "Anomaly on " + s.NodeID + " (" + s.Metric + ")"
	}
}

func clusterTitle(metric string) string {
	return "Cluster-wide " + metric + " pressure"
}

func clusterDescription(metric string, nodes int) string {
	return fmt.Sprintf("Sustained %s anomaly across %d distinct nodes within the correlation window", metric, nodes)
}

func mergeDescription(existing string, s Signal) string {
	if s.Kind == "anomaly" {
		suffix := s.NodeID + " " + s.Metric
		if strings.Contains(existing, suffix) {
			return existing
		}
		if existing == "" {
			return suffix
		}
		return existing + ", " + suffix
	}
	parts := strings.Split(existing, ", ")
	for _, p := range parts {
		if p == s.Kind {
			return existing
		}
	}
	return strings.TrimSpace(existing + ", " + s.Kind)
}

func severityOf(v interface{}) string {
	s := strings.ToLower(fmt.Sprint(v))
	switch s {
	case "critical":
		return SeverityCritical
	case "warning":
		return SeverityWarning
	default:
		return SeverityWarning
	}
}

func floatOf(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}

func dedupe(in []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
package ai

import (
	"testing"
	"time"

	"github.com/atlas-engine/atlas/internal/incident"
	"github.com/atlas-engine/atlas/internal/models"
)

type fakeSnapshot struct {
	nodes    map[string]*models.Node
	health   float64
	bottles  []string
}

func (f *fakeSnapshot) GetNode(id string) (*models.Node, bool) {
	n, ok := f.nodes[id]
	return n, ok
}
func (f *fakeSnapshot) HealthScore() float64    { return f.health }
func (f *fakeSnapshot) Bottlenecks() []string   { return f.bottles }

func viewFor(signals ...incident.Signal) *incident.View {
	return &incident.View{Severity: incident.SeverityCritical, Signals: signals}
}

func TestChaosArchetypeWins(t *testing.T) {
	ad := NewAdvisor(3)
	v := viewFor(incident.Signal{Kind: "chaos", NodeID: "exp-1", Severity: incident.SeverityWarning, Time: time.Now()})

	a := ad.Analyze(v, v.Signals, &fakeSnapshot{})
	if a.Archetype != ArchetypeChaosInjection {
		t.Fatalf("expected chaos archetype, got %s", a.Archetype)
	}
	if len(a.Recommendations) == 0 {
		t.Fatalf("expected recommendations")
	}
	for _, r := range a.Recommendations {
		if !r.RequiresApproval {
			t.Fatalf("every recommendation must require approval: %+v", r)
		}
	}
}

func TestClusterPressureArchetype(t *testing.T) {
	ad := NewAdvisor(3)
	signals := []incident.Signal{
		{Kind: "anomaly", NodeID: "n1", Metric: "cpu", Severity: incident.SeverityWarning},
		{Kind: "anomaly", NodeID: "n2", Metric: "cpu", Severity: incident.SeverityWarning},
		{Kind: "anomaly", NodeID: "n3", Metric: "cpu", Severity: incident.SeverityWarning},
	}
	v := &incident.View{Signals: signals, AffectedNodes: []string{"n1", "n2", "n3"}}
	snap := &fakeSnapshot{nodes: map[string]*models.Node{
		"n1": {ID: "n1", State: models.NodeStateHealthy, Health: 0.9},
		"n2": {ID: "n2", State: models.NodeStateHealthy, Health: 0.9},
		"n3": {ID: "n3", State: models.NodeStateHealthy, Health: 0.9},
	}}

	a := ad.Analyze(v, v.Signals, snap)
	if a.Archetype != ArchetypeClusterPressure {
		t.Fatalf("expected cluster pressure, got %s", a.Archetype)
	}
}

func TestSingleFailureArchetype(t *testing.T) {
	ad := NewAdvisor(3)
	signals := []incident.Signal{{Kind: "node_failed", NodeID: "db-1", Severity: incident.SeverityCritical}}
	v := &incident.View{Signals: signals, AffectedNodes: []string{"db-1"}}
	snap := &fakeSnapshot{nodes: map[string]*models.Node{
		"db-1": {ID: "db-1", State: models.NodeStateFailed, Health: 0},
	}}

	a := ad.Analyze(v, v.Signals, snap)
	if a.Archetype != ArchetypeSingleFailure {
		t.Fatalf("expected single failure, got %s", a.Archetype)
	}
}

func TestCapacityBottleneckArchetype(t *testing.T) {
	ad := NewAdvisor(3)
	signals := []incident.Signal{{Kind: "anomaly", NodeID: "web-1", Metric: "memory", Severity: incident.SeverityWarning}}
	v := &incident.View{Signals: signals, AffectedNodes: []string{"web-1"}}
	snap := &fakeSnapshot{nodes: map[string]*models.Node{
		"web-1": {ID: "web-1", State: models.NodeStateDegraded, Health: 0.3},
	}}

	a := ad.Analyze(v, v.Signals, snap)
	if a.Archetype != ArchetypeCapacityBottleneck {
		t.Fatalf("expected capacity bottleneck, got %s", a.Archetype)
	}
}

func TestNilSnapshotSafe(t *testing.T) {
	ad := NewAdvisor(3)
	a := ad.Analyze(nil, nil, nil)
	if a == nil || a.Archetype != ArchetypeIndeterminate {
		t.Fatalf("nil input should yield indefinite analysis, got %+v", a)
	}
}
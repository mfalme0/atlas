package incident

import (
	"testing"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
)

func TestSignalOpensIncident(t *testing.T) {
	m := NewManager(Options{})
	m.Start()

	m.RecordSignal(Signal{Kind: "anomaly", NodeID: "db-1", Metric: "cpu", Severity: SeverityCritical, Time: time.Now()})

	list := m.List(false)
	if len(list) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(list))
	}
	inc := list[0]
	if inc.State != StateOpen || inc.Severity != SeverityCritical {
		t.Fatalf("unexpected incident: %+v", inc)
	}
	if len(inc.AffectedNodes) != 1 || inc.AffectedNodes[0] != "db-1" {
		t.Fatalf("affected nodes wrong: %v", inc.AffectedNodes)
	}
}

func TestCoalesceWithinWindow(t *testing.T) {
	clock := time.Now
	m := NewManager(Options{Clock: clock})

	m.RecordSignal(Signal{Kind: "anomaly", NodeID: "db-1", Metric: "cpu", Severity: SeverityWarning, Time: clock()})
	m.RecordSignal(Signal{Kind: "anomaly", NodeID: "db-1", Metric: "mem", Severity: SeverityWarning, Time: clock()})
	m.RecordSignal(Signal{Kind: "anomaly", NodeID: "db-1", Metric: "disk", Severity: SeverityCritical, Time: clock()})

	list := m.List(false)
	if len(list) != 1 {
		t.Fatalf("expected 1 coalesced incident, got %d", len(list))
	}
	if list[0].Severity != SeverityCritical {
		t.Fatalf("severity should escalate to critical, got %s", list[0].Severity)
	}
	if len(list[0].Signals) != 3 {
		t.Fatalf("expected 3 coalesced signals, got %d", len(list[0].Signals))
	}
}

func TestClusterWideIncident(t *testing.T) {
	m := NewManager(Options{ClusterThreshold: 3, Window: time.Minute})

	m.RecordSignal(Signal{Kind: "anomaly", NodeID: "n1", Metric: "cpu", Severity: SeverityWarning, Time: time.Now()})
	m.RecordSignal(Signal{Kind: "anomaly", NodeID: "n2", Metric: "cpu", Severity: SeverityWarning, Time: time.Now()})
	m.RecordSignal(Signal{Kind: "anomaly", NodeID: "n3", Metric: "cpu", Severity: SeverityWarning, Time: time.Now()})

	list := m.List(true)
	for _, v := range list {
		t.Logf("incident %s title=%q affected=%v", v.ID, v.Title, v.AffectedNodes)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 cluster incident, got %d", len(list))
	}
	if list[0].Severity != SeverityCritical {
		t.Fatalf("cluster incidents should be critical, got %s", list[0].Severity)
	}
	if len(list[0].AffectedNodes) != 3 {
		t.Fatalf("expected 3 affected nodes, got %v", list[0].AffectedNodes)
	}
}

func TestLifecycleResolve(t *testing.T) {
	m := NewManager(Options{})
	m.RecordSignal(Signal{Kind: "node_failed", NodeID: "api-1", Severity: SeverityCritical, Time: time.Now()})

	id := m.List(false)[0].ID
	if !m.Investigate(id) {
		t.Fatalf("investigate failed")
	}
	if m.Get(id).State != StateInvestigating {
		t.Fatalf("expected investigating state")
	}
	if !m.Resolve(id, "restarted upstream") {
		t.Fatalf("resolve failed")
	}
	inc := m.Get(id)
	if inc.State != StateResolved || inc.ResolvedAt == nil || inc.Resolution != "restarted upstream" {
		t.Fatalf("resolve not applied: %+v", inc)
	}
	if m.Get("nope") != nil {
		t.Fatalf("expected nil for unknown id")
	}
	if len(m.List(true)) != 0 {
		t.Fatalf("expected no open incidents after resolve")
	}
}

func TestSubscribeFromBus(t *testing.T) {
	bus := event.NewBus(8)
	bus.Start()
	m := NewManager(Options{Bus: bus})
	m.Start()

	bus.Publish(event.NewEvent(event.NodeFailed, "test", map[string]interface{}{"node_id": "cache-1"}))
	time.Sleep(50 * time.Millisecond)

	list := m.List(false)
	if len(list) != 1 {
		t.Fatalf("expected 1 incident from bus, got %d", len(list))
	}
	if list[0].Title != "Node failed: cache-1" {
		t.Fatalf("unexpected title: %s", list[0].Title)
	}
}
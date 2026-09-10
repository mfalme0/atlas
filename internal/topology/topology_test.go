package topology

import (
	"testing"

	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/models"
)

func newTestEngine() *Engine {
	bus := event.NewBus(100)
	bus.Start()
	return NewEngine(bus)
}

func TestAddNode(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "web", Name: "Web Server", Type: models.NodeTypeApp})

	n, ok := e.GetNode("web")
	if !ok || n.Name != "Web Server" {
		t.Error("expected to find web node")
	}
	if len(e.Nodes()) != 1 {
		t.Errorf("expected 1 node, got %d", len(e.Nodes()))
	}
}

func TestRemoveNode(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A"})
	e.AddNode(&models.Node{ID: "b", Name: "B"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "a", Target: "b", Type: models.EdgeDependsOn})

	e.RemoveNode("a")
	if _, ok := e.GetNode("a"); ok {
		t.Error("expected node a removed")
	}
}

func TestHasNode(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "x", Name: "X"})
	if _, ok := e.GetNode("x"); !ok {
		t.Error("expected GetNode to return true")
	}
}

func TestAddEdge(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "db", Name: "DB"})
	e.AddNode(&models.Node{ID: "api", Name: "API"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "api", Target: "db", Type: models.EdgeDependsOn, Weight: 1})

	edges := e.Edges()
	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(edges))
	}
}

func TestDirectDependents(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "db", Name: "DB"})
	e.AddNode(&models.Node{ID: "api", Name: "API"})
	e.AddNode(&models.Node{ID: "web", Name: "Web"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "api", Target: "db", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e2", Source: "web", Target: "api", Type: models.EdgeDependsOn})

	direct := e.DirectDependents("db")
	if len(direct) != 1 || direct[0] != "api" {
		t.Errorf("expected [api], got %v", direct)
	}
}

func TestIndirectDependents(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "db", Name: "DB"})
	e.AddNode(&models.Node{ID: "api", Name: "API"})
	e.AddNode(&models.Node{ID: "web", Name: "Web"})
	e.AddNode(&models.Node{ID: "mobile", Name: "Mobile"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "api", Target: "db", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e2", Source: "web", Target: "api", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e3", Source: "mobile", Target: "api", Type: models.EdgeDependsOn})

	indirect := e.IndirectDependents("db")
	if len(indirect) != 3 {
		t.Errorf("expected 3 indirect dependents, got %d: %v", len(indirect), indirect)
	}
}

func TestImpactAnalysis(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "db", Name: "DB"})
	e.AddNode(&models.Node{ID: "api", Name: "API"})
	e.AddNode(&models.Node{ID: "web", Name: "Web"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "api", Target: "db", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e2", Source: "web", Target: "api", Type: models.EdgeDependsOn})

	impact := e.ImpactAnalysis("db")
	if impact.AffectedCount != 3 {
		t.Errorf("expected 3 affected, got %d", impact.AffectedCount)
	}
	if len(impact.DirectDependents) != 1 {
		t.Errorf("expected 1 direct dependent, got %d", len(impact.DirectDependents))
	}
	if len(impact.IndirectDependents) != 2 {
		t.Errorf("expected 2 indirect dependents, got %d", len(impact.IndirectDependents))
	}
	if impact.Criticality <= 0 || impact.Criticality > 1 {
		t.Errorf("expected criticality in (0, 1], got %f", impact.Criticality)
	}
}

func TestBottlenecks(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A"})
	e.AddNode(&models.Node{ID: "b", Name: "B"})
	e.AddNode(&models.Node{ID: "c", Name: "C"})
	e.AddNode(&models.Node{ID: "d", Name: "D"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "a", Target: "b", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e2", Source: "b", Target: "c", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e3", Source: "b", Target: "d", Type: models.EdgeDependsOn})

	bn := e.Bottlenecks()
	if len(bn) != 1 || bn[0] != "b" {
		t.Errorf("expected [b], got %v", bn)
	}
}

func TestRecoveryPath(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A"})
	e.AddNode(&models.Node{ID: "b", Name: "B"})
	e.AddNode(&models.Node{ID: "c", Name: "C"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "a", Target: "b", Type: models.EdgeDependsOn, Weight: 1})
	e.AddEdge(&models.Edge{ID: "e2", Source: "b", Target: "c", Type: models.EdgeDependsOn, Weight: 1})

	path, dist := e.RecoveryPath("a", "c")
	if len(path) != 3 {
		t.Errorf("expected path of length 3, got %d", len(path))
	}
	if dist != 2.0 {
		t.Errorf("expected distance 2.0, got %f", dist)
	}
}

func TestCycles(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A"})
	e.AddNode(&models.Node{ID: "b", Name: "B"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "a", Target: "b", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e2", Source: "b", Target: "a", Type: models.EdgeDependsOn})

	if !e.Cycles() {
		t.Error("expected cycle detected")
	}
}

func TestNoCycles(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A"})
	e.AddNode(&models.Node{ID: "b", Name: "B"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "a", Target: "b", Type: models.EdgeDependsOn})

	if e.Cycles() {
		t.Error("expected no cycles")
	}
}

func TestTopologicalOrder(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A"})
	e.AddNode(&models.Node{ID: "b", Name: "B"})
	e.AddNode(&models.Node{ID: "c", Name: "C"})
	e.AddEdge(&models.Edge{ID: "e1", Source: "a", Target: "b", Type: models.EdgeDependsOn})
	e.AddEdge(&models.Edge{ID: "e2", Source: "b", Target: "c", Type: models.EdgeDependsOn})

	order := e.TopologicalOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
	idx := make(map[string]int)
	for i, id := range order {
		idx[id] = i
	}
	if idx["a"] > idx["b"] || idx["b"] > idx["c"] {
		t.Error("wrong topological order")
	}
}

func TestHealthScore(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A", Health: 1.0})
	e.AddNode(&models.Node{ID: "b", Name: "B", Health: 0.8})

	score := e.HealthScore()
	if score != 0.9 {
		t.Errorf("expected 0.9, got %f", score)
	}
}

func TestSnapshot(t *testing.T) {
	e := newTestEngine()
	e.AddNode(&models.Node{ID: "a", Name: "A"})

	snap := e.Snapshot()
	if len(snap.Nodes) != 1 {
		t.Errorf("expected 1 node in snapshot, got %d", len(snap.Nodes))
	}
}

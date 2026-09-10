package chaos

import (
	"context"
	"testing"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/models"
	"github.com/atlas-engine/atlas/internal/topology"
)

func testTopology() *topology.Engine {
	topo := topology.NewEngine(event.NewBus(16))
	topo.AddNode(&models.Node{ID: "web", Name: "web", Type: models.NodeTypeApp, Health: 1.0, State: models.NodeStateHealthy})
	topo.AddNode(&models.Node{ID: "db", Name: "db", Type: models.NodeTypeDatabase, Health: 1.0, State: models.NodeStateHealthy})
	topo.AddNode(&models.Node{ID: "cache", Name: "cache", Type: models.NodeTypeDatabase, Health: 1.0, State: models.NodeStateHealthy})
	topo.AddEdge(&models.Edge{ID: "e1", Source: "web", Target: "db", Type: models.EdgeDependsOn, Weight: 1})
	topo.AddEdge(&models.Edge{ID: "e2", Source: "cache", Target: "db", Type: models.EdgeDependsOn, Weight: 1})
	return topo
}

func newEngine(topo *topology.Engine) *Engine {
	return NewEngine(topo, event.NewBus(16), nil)
}

func TestKillNodeReverts(t *testing.T) {
	topo := testTopology()
	e := newEngine(topo)

	exp := &Experiment{ID: "exp-1", Type: TypeKillNode, Target: "db", Duration: time.Minute}
	if err := e.Run(context.Background(), exp); err != nil {
		t.Fatalf("run: %v", err)
	}

	node, _ := topo.GetNode("db")
	if node.Health != 0 || node.State != models.NodeStateFailed {
		t.Fatalf("expected failed state after kill, got health=%v state=%v", node.Health, node.State)
	}
	if exp.Impact == nil || exp.Impact.AffectedCount != 2 {
		t.Fatalf("expected 2 affected dependents, got %+v", exp.Impact)
	}

	e.Stop(exp.ID)

	node, _ = topo.GetNode("db")
	if node.Health != 1.0 || node.State != models.NodeStateHealthy {
		t.Fatalf("expected healthy state after revert, got health=%v state=%v", node.Health, node.State)
	}
	if exp.Status != StatusCompleted {
		t.Fatalf("expected completed status, got %s", exp.Status)
	}

	if len(e.Active()) != 0 {
		t.Fatalf("expected no active experiments")
	}
	if len(e.Results()) != 1 {
		t.Fatalf("expected 1 result, got %d", len(e.Results()))
	}
}

func TestNetworkPartitionReverts(t *testing.T) {
	topo := testTopology()
	e := newEngine(topo)

	exp := &Experiment{ID: "exp-2", Type: TypeNetworkPartition, Target: "db", Duration: time.Minute}
	if err := e.Run(context.Background(), exp); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := len(topo.Edges()); got != 0 {
		t.Fatalf("expected 0 edges during partition, got %d", got)
	}

	e.Stop(exp.ID)

	if got := len(topo.Edges()); got != 2 {
		t.Fatalf("expected 2 edges restored, got %d", got)
	}
}

func TestLatencyDegradesHealth(t *testing.T) {
	topo := testTopology()
	e := newEngine(topo)

	exp := &Experiment{ID: "exp-3", Type: TypeLatency, Target: "cache", Duration: time.Minute}
	if err := e.Run(context.Background(), exp); err != nil {
		t.Fatalf("run: %v", err)
	}

	node, _ := topo.GetNode("cache")
	if node.Health != 0.35 {
		t.Fatalf("expected degraded health 0.35, got %v", node.Health)
	}

	e.Stop(exp.ID)

	node, _ = topo.GetNode("cache")
	if node.Health != 1.0 {
		t.Fatalf("expected health restored, got %v", node.Health)
	}
}

func TestAutoRevertAfterDuration(t *testing.T) {
	topo := testTopology()
	e := newEngine(topo)

	exp := &Experiment{ID: "exp-4", Type: TypeKillNode, Target: "db", Duration: 50 * time.Millisecond}
	if err := e.Run(context.Background(), exp); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(e.Active()) != 1 {
		t.Fatalf("expected experiment active")
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(e.Active()) == 0 && exp.Status == StatusCompleted {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(e.Active()) != 0 {
		t.Fatalf("expected auto-revert, got %d active", len(e.Active()))
	}
	if exp.Status != StatusCompleted {
		t.Fatalf("expected completed status after auto-revert, got %s", exp.Status)
	}

	node, _ := topo.GetNode("db")
	if node.Health != 1.0 {
		t.Fatalf("expected health restored after auto-revert, got %v", node.Health)
	}
}

func TestUnknownTypeFails(t *testing.T) {
	topo := testTopology()
	e := newEngine(topo)

	exp := &Experiment{ID: "exp-5", Type: Type("nonsense"), Target: "db"}
	if err := e.Run(context.Background(), exp); err == nil {
		t.Fatalf("expected error for unknown type")
	}
	if exp.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", exp.Status)
	}
	if len(e.Active()) != 0 {
		t.Fatalf("expected no active experiments")
	}
}

func TestMissingTargetFails(t *testing.T) {
	topo := testTopology()
	e := newEngine(topo)

	exp := &Experiment{ID: "exp-6", Type: TypeKillNode, Target: "missing"}
	if err := e.Run(context.Background(), exp); err == nil {
		t.Fatalf("expected error for missing target")
	}
}

func TestRunTwiceRejected(t *testing.T) {
	topo := testTopology()
	e := newEngine(topo)

	exp := &Experiment{ID: "exp-7", Type: TypeKillNode, Target: "db", Duration: time.Minute}
	if err := e.Run(context.Background(), exp); err != nil {
		t.Fatalf("first run: %v", err)
	}
	e.Stop(exp.ID)

	exp2 := &Experiment{ID: "exp-7", Type: TypeLatency, Target: "db", Duration: time.Minute}
	if err := e.Run(context.Background(), exp2); err == nil {
		t.Fatalf("expected second run of same ID to be rejected")
	}
}
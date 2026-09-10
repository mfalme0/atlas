package scheduler

import (
	"testing"

	"github.com/atlas-engine/atlas/internal/event"
)

func newTestScheduler() *Engine {
	bus := event.NewBus(100)
	bus.Start()
	return NewEngine(bus)
}

func TestScheduleBasic(t *testing.T) {
	s := newTestScheduler()
	s.RegisterNode(&NodeInfo{ID: "n1", CPUCapacity: 8, MemoryCapacity: 16, Health: 1.0})
	s.RegisterNode(&NodeInfo{ID: "n2", CPUCapacity: 4, MemoryCapacity: 8, Health: 1.0})

	w := &Workload{ID: "w1", CPU: 2, Memory: 4, Priority: 5}
	result := s.Schedule(w)
	if result == nil {
		t.Fatal("expected scheduling result")
	}
	if result.NodeID != "n1" {
		t.Errorf("expected n1 (more resources), got %s", result.NodeID)
	}
	if result.Score <= 0 {
		t.Errorf("expected positive score, got %f", result.Score)
	}
}

func TestScheduleUnhealthyNode(t *testing.T) {
	s := newTestScheduler()
	s.RegisterNode(&NodeInfo{ID: "n1", CPUCapacity: 8, Health: 0.0})
	s.RegisterNode(&NodeInfo{ID: "n2", CPUCapacity: 4, Health: 1.0})

	w := &Workload{ID: "w1", CPU: 1}
	result := s.Schedule(w)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.NodeID != "n2" {
		t.Errorf("expected n2 (healthy), got %s", result.NodeID)
	}
}

func TestScheduleScoreBreakdown(t *testing.T) {
	s := newTestScheduler()
	s.RegisterNode(&NodeInfo{ID: "n1", CPUCapacity: 8, MemoryCapacity: 16, Health: 1.0})

	w := &Workload{ID: "w1", CPU: 2, Memory: 4, Priority: 5}
	result := s.Schedule(w)
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Breakdown.CPUScore <= 0 {
		t.Error("expected positive CPU score")
	}
	if result.Breakdown.MemoryScore <= 0 {
		t.Error("expected positive memory score")
	}
}

func TestScheduleUtilizationPenalty(t *testing.T) {
	s := newTestScheduler()
	s.RegisterNode(&NodeInfo{ID: "n1", CPUCapacity: 8, MemoryCapacity: 16, CPUUsed: 7, MemoryUsed: 14, Health: 1.0})
	s.RegisterNode(&NodeInfo{ID: "n2", CPUCapacity: 8, MemoryCapacity: 16, Health: 1.0})

	w := &Workload{ID: "w1", CPU: 1, Memory: 2}
	result := s.Schedule(w)
	if result.NodeID != "n2" {
		t.Errorf("expected n2 (less utilized), got %s", result.NodeID)
	}
}

func TestScheduleAll(t *testing.T) {
	s := newTestScheduler()
	s.RegisterNode(&NodeInfo{ID: "n1", CPUCapacity: 8, MemoryCapacity: 16, Health: 1.0})
	s.RegisterNode(&NodeInfo{ID: "n2", CPUCapacity: 8, MemoryCapacity: 16, Health: 1.0})

	workloads := []*Workload{
		{ID: "w1", CPU: 2, Memory: 4},
		{ID: "w2", CPU: 2, Memory: 4},
		{ID: "w3", CPU: 2, Memory: 4},
	}
	results := s.ScheduleAll(workloads)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestScheduleEmpty(t *testing.T) {
	s := newTestScheduler()
	w := &Workload{ID: "w1", CPU: 1}
	result := s.Schedule(w)
	if result != nil {
		t.Error("expected nil for no nodes")
	}
}

func TestNodes(t *testing.T) {
	s := newTestScheduler()
	s.RegisterNode(&NodeInfo{ID: "n1"})
	s.RegisterNode(&NodeInfo{ID: "n2"})

	if len(s.Nodes()) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(s.Nodes()))
	}
}

func BenchmarkSchedule(b *testing.B) {
	s := newTestScheduler()
	for i := 0; i < 100; i++ {
		s.RegisterNode(&NodeInfo{
			ID:            string(rune('a' + i%26)),
			CPUCapacity:   16,
			MemoryCapacity: 64,
			Health:        1.0,
		})
	}
	w := &Workload{ID: "w1", CPU: 4, Memory: 8, Priority: 5}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Schedule(w)
	}
}

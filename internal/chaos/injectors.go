package chaos

import (
	"context"

	"github.com/atlas-engine/atlas/internal/models"
	"github.com/atlas-engine/atlas/internal/topology"
)

// impactOf computes the topology blast radius for a target node.
func impactOf(topo *topology.Engine, target string) *Impact {
	result := topo.ImpactAnalysis(target)
	distinct := make(map[string]bool)
	for _, id := range result.DirectDependents {
		distinct[id] = true
	}
	for _, id := range result.IndirectDependents {
		distinct[id] = true
	}
	return &Impact{
		DirectDependents:   result.DirectDependents,
		IndirectDependents: result.IndirectDependents,
		AffectedCount:      len(distinct),
		Criticality:        result.Criticality,
	}
}

type nodeSnapshot struct {
	state  models.NodeState
	health float64
	load   float64
	edges  []*models.Edge
}

func snapshotNode(topo *topology.Engine, id string) (*nodeSnapshot, bool) {
	node, ok := topo.GetNode(id)
	if !ok {
		return nil, false
	}
	var edges []*models.Edge
	for _, ed := range topo.Edges() {
		if ed.Source == id || ed.Target == id {
			edges = append(edges, ed)
		}
	}
	return &nodeSnapshot{
		state:  node.State,
		health: node.Health,
		load:   node.Load,
		edges:  edges,
	}, true
}

// NodeKillInjector marks a node as failed (state=failed, health=0) and reverts it.
type NodeKillInjector struct{}

func (i *NodeKillInjector) Type() Type { return TypeKillNode }

func (i *NodeKillInjector) Apply(ctx context.Context, e *Experiment, topo *topology.Engine) (*Impact, error) {
	if _, ok := topo.GetNode(e.Target); !ok {
		return nil, errTargetMissing(e.Target)
	}
	topo.SetNodeState(e.Target, models.NodeStateFailed)
	topo.SetNodeHealth(e.Target, 0)
	return impactOf(topo, e.Target), nil
}

func (i *NodeKillInjector) Revert(e *Experiment, topo *topology.Engine) error {
	topo.SetNodeState(e.Target, models.NodeStateHealthy)
	topo.SetNodeHealth(e.Target, 1.0)
	return nil
}

// NetworkPartitionInjector disconnects a node from the graph by removing all of
// its edges, then restores them on revert.
type NetworkPartitionInjector struct{}

func (i *NetworkPartitionInjector) Type() Type { return TypeNetworkPartition }

func (i *NetworkPartitionInjector) Apply(ctx context.Context, e *Experiment, topo *topology.Engine) (*Impact, error) {
	snap, ok := snapshotNode(topo, e.Target)
	if !ok {
		return nil, errTargetMissing(e.Target)
	}
	for _, ed := range snap.edges {
		topo.RemoveEdge(ed.ID)
	}
	if e.Meta == nil {
		e.Meta = make(map[string]interface{})
	}
	e.Meta["edges"] = snap.edges
	return &Impact{AffectedCount: len(snap.edges)}, nil
}

func (i *NetworkPartitionInjector) Revert(e *Experiment, topo *topology.Engine) error {
	edges, _ := e.Meta["edges"].([]*models.Edge)
	for _, ed := range edges {
		topo.AddEdge(ed)
	}
	return nil
}

// LatencyInjector degrades a node's health to simulate network latency, then
// restores the original health on revert.
type LatencyInjector struct{}

func (i *LatencyInjector) Type() Type { return TypeLatency }

func (i *LatencyInjector) Apply(ctx context.Context, e *Experiment, topo *topology.Engine) (*Impact, error) {
	node, ok := topo.GetNode(e.Target)
	if !ok {
		return nil, errTargetMissing(e.Target)
	}
	_ = node
	topo.SetNodeState(e.Target, models.NodeStateDegraded)
	topo.SetNodeHealth(e.Target, 0.35)
	return impactOf(topo, e.Target), nil
}

func (i *LatencyInjector) Revert(e *Experiment, topo *topology.Engine) error {
	topo.SetNodeState(e.Target, models.NodeStateHealthy)
	topo.SetNodeHealth(e.Target, 1.0)
	return nil
}

// CPUStressInjector models a CPU spike by degrading health.
type CPUStressInjector struct{}

func (i *CPUStressInjector) Type() Type { return TypeCPUStress }

func (i *CPUStressInjector) Apply(ctx context.Context, e *Experiment, topo *topology.Engine) (*Impact, error) {
	if _, ok := topo.GetNode(e.Target); !ok {
		return nil, errTargetMissing(e.Target)
	}
	topo.SetNodeState(e.Target, models.NodeStateDegraded)
	topo.SetNodeHealth(e.Target, 0.55)
	return impactOf(topo, e.Target), nil
}

func (i *CPUStressInjector) Revert(e *Experiment, topo *topology.Engine) error {
	topo.SetNodeState(e.Target, models.NodeStateHealthy)
	topo.SetNodeHealth(e.Target, 1.0)
	return nil
}

// MemoryPressureInjector models memory pressure by degrading health.
type MemoryPressureInjector struct{}

func (i *MemoryPressureInjector) Type() Type { return TypeMemoryPressure }

func (i *MemoryPressureInjector) Apply(ctx context.Context, e *Experiment, topo *topology.Engine) (*Impact, error) {
	if _, ok := topo.GetNode(e.Target); !ok {
		return nil, errTargetMissing(e.Target)
	}
	topo.SetNodeState(e.Target, models.NodeStateDegraded)
	topo.SetNodeHealth(e.Target, 0.45)
	return impactOf(topo, e.Target), nil
}

func (i *MemoryPressureInjector) Revert(e *Experiment, topo *topology.Engine) error {
	topo.SetNodeState(e.Target, models.NodeStateHealthy)
	topo.SetNodeHealth(e.Target, 1.0)
	return nil
}

func errTargetMissing(target string) error {
	return &targetError{target: target}
}

type targetError struct {
	target string
}

func (e *targetError) Error() string {
	return "chaos: target not found: " + e.target
}
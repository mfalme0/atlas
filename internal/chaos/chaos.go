// Package chaos implements a controlled failure-injection engine.
//
// Experiments deliberately degrade or disconnect infrastructure targets so that
// operators and automated detection can validate resilience. Every experiment:
//   - Is planned against the topology (impact analysis estimates blast radius).
//   - Applies a fault through an injector and records the topology impact.
//   - Reverts the fault automatically after its Duration, or when aborted.
//   - Emits chaos.started / chaos.completed events on the event bus.
package chaos

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/topology"
)

// Type identifies a chaos experiment kind.
type Type string

const (
	TypeKillNode          Type = "kill_node"
	TypeNetworkPartition  Type = "network_partition"
	TypeLatency           Type = "latency"
	TypeCPUStress         Type = "cpu_stress"
	TypeMemoryPressure    Type = "memory_pressure"
)

// Status is the lifecycle state of an experiment.
type Status string

const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusCompleted Status = "completed"
	StatusAborted  Status = "aborted"
	StatusFailed   Status = "failed"
)

// Experiment describes a chaos injection.
type Experiment struct {
	ID        string     `json:"id"`
	Type      Type       `json:"type"`
	Target    string     `json:"target"`
	Intensity float64    `json:"intensity"`
	Duration  time.Duration `json:"duration"`
	Label     string     `json:"label,omitempty"`
	Meta      map[string]interface{} `json:"-"`

	Status     Status       `json:"status"`
	Impact     *Impact      `json:"impact,omitempty"`
	StartedAt  time.Time    `json:"started_at"`
	EndedAt    time.Time    `json:"ended_at"`
	Error      string       `json:"error,omitempty"`
}

// Impact estimates the blast radius of an experiment against the topology.
type Impact struct {
	DirectDependents  []string `json:"direct_dependents"`
	IndirectDependents []string `json:"indirect_dependents"`
	AffectedCount     int      `json:"affected_count"`
	Criticality       float64  `json:"criticality"`
}

// Injector applies and reverts a specific fault on the topology.
type Injector interface {
	Type() Type
	Apply(ctx context.Context, e *Experiment, topo *topology.Engine) (*Impact, error)
	Revert(e *Experiment, topo *topology.Engine) error
}

// Engine runs chaos experiments against the topology.
type Engine struct {
	mu        sync.RWMutex
	topo      *topology.Engine
	bus       *event.Bus
	logger    *slog.Logger
	injectors map[Type]Injector
	active    map[string]context.CancelFunc
	running   map[string]*Experiment
	results   []*Experiment
	acked     map[string]bool
}

// NewEngine creates a chaos engine with all built-in injectors.
func NewEngine(topo *topology.Engine, bus *event.Bus, logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	e := &Engine{
		topo:      topo,
		bus:       bus,
		logger:    logger,
		injectors: make(map[Type]Injector),
		active:    make(map[string]context.CancelFunc),
		running:   make(map[string]*Experiment),
		results:   make([]*Experiment, 0),
		acked:     make(map[string]bool),
	}
	injectors := []Injector{
		&NodeKillInjector{},
		&NetworkPartitionInjector{},
		&LatencyInjector{},
		&CPUStressInjector{},
		&MemoryPressureInjector{},
	}
	for _, inj := range injectors {
		e.injectors[inj.Type()] = inj
	}
	return e
}

// Run injects a fault. The fault reverts automatically after Duration unless
// the experiment is aborted first. Returns ErrUnknownType for unsupported types.
func (e *Engine) Run(ctx context.Context, exp *Experiment) error {
	if exp.ID == "" {
		return fmt.Errorf("chaos: experiment requires an id")
	}
	inj, ok := e.injectors[exp.Type]
	if !ok {
		exp.Status = StatusFailed
		exp.Error = fmt.Sprintf("chaos: unknown experiment type %q", exp.Type)
		exp.EndedAt = time.Now()
		e.mu.Lock()
		e.results = append(e.results, exp)
		e.mu.Unlock()
		return fmt.Errorf("%s", exp.Error)
	}

	e.mu.Lock()
	if e.acked[exp.ID] {
		e.mu.Unlock()
		return fmt.Errorf("chaos: experiment %q already ran", exp.ID)
	}
	e.acked[exp.ID] = true
	e.mu.Unlock()

	impact, err := inj.Apply(ctx, exp, e.topo)
	if err != nil {
		err = fmt.Errorf("chaos: apply %s to %s: %w", exp.Type, exp.Target, err)
		exp.Status = StatusFailed
		exp.Error = err.Error()
		exp.EndedAt = time.Now()
		e.mu.Lock()
		e.results = append(e.results, exp)
		e.mu.Unlock()
		return err
	}

	duration := exp.Duration
	if duration <= 0 {
		duration = 30 * time.Second
	}

	runCtx, cancel := context.WithCancel(ctx)
	e.logger.Info("chaos experiment started",
		slog.String("id", exp.ID),
		slog.String("type", string(exp.Type)),
		slog.String("target", exp.Target),
		slog.Duration("duration", duration))

	exp.Status = StatusRunning
	exp.Impact = impact
	exp.StartedAt = time.Now()
	e.emit(event.ChaosExperimentStarted, exp)

	e.mu.Lock()
	e.active[exp.ID] = cancel
	e.running[exp.ID] = exp
	e.mu.Unlock()

	go func() {
		select {
		case <-runCtx.Done():
		case <-time.After(duration):
		}
		e.Stop(exp.ID)
	}()

	return nil
}

// Stop reverts a running experiment, or records an aborted one.
func (e *Engine) Stop(id string) {
	e.mu.Lock()
	cancel, ok := e.active[id]
	if !ok {
		e.mu.Unlock()
		return
	}
	delete(e.active, id)
	exp, exists := e.running[id]
	if exists {
		delete(e.running, id)
	}
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if !exists {
		return
	}

	inj := e.injectors[exp.Type]
	if inj != nil {
		if err := inj.Revert(exp, e.topo); err != nil {
			e.logger.Error("chaos experiment revert failed",
				slog.String("id", id),
				slog.Any("error", err))
		}
	}

	if exp.Status == StatusRunning {
		exp.Status = StatusCompleted
	} else {
		exp.Status = StatusAborted
	}
	exp.EndedAt = time.Now()

	e.mu.Lock()
	e.results = append(e.results, exp)
	e.mu.Unlock()

	e.logger.Info("chaos experiment finished",
		slog.String("id", id),
		slog.String("status", string(exp.Status)))
	e.emit(event.ChaosExperimentCompleted, exp)
}

// Active returns a snapshot of currently running experiments.
func (e *Engine) Active() []*Experiment {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*Experiment, 0, len(e.running))
	for _, exp := range e.running {
		result = append(result, exp)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})
	return result
}

// Results returns all completed/failed experiments.
func (e *Engine) Results() []*Experiment {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*Experiment, len(e.results))
	copy(result, e.results)
	return result
}

func (e *Engine) emit(t event.EventType, exp *Experiment) {
	if e.bus == nil {
		return
	}
	data := map[string]interface{}{
		"experiment_id": exp.ID,
		"type":          string(exp.Type),
		"target":        exp.Target,
		"status":        string(exp.Status),
	}
	if exp.Impact != nil {
		data["affected_count"] = exp.Impact.AffectedCount
		data["criticality"] = exp.Impact.Criticality
	}
	e.bus.Publish(event.NewEvent(t, "chaos", data))
}
// Package scheduler implements a workload scheduler with resource scoring.
//
// The scheduler evaluates nodes against workload requirements and produces
// a scored ranking with transparent reasoning behind each decision.
package scheduler

import (
	"sort"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
)

// Workload represents a job to be scheduled.
type Workload struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	CPU              float64           `json:"cpu"`
	Memory           float64           `json:"memory"`
	GPU              float64           `json:"gpu"`
	NetworkBandwidth float64           `json:"network_bandwidth"`
	LatencyMax       float64           `json:"latency_max"`
	Priority         int               `json:"priority"`
	Affinity         []string          `json:"affinity"`
	AntiAffinity     []string          `json:"anti_affinity"`
	Labels           map[string]string `json:"labels"`
	CreatedAt        time.Time         `json:"created_at"`
}

// NodeInfo represents a scheduling target.
type NodeInfo struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	CPUCapacity    float64           `json:"cpu_capacity"`
	MemoryCapacity float64           `json:"memory_capacity"`
	GPUCapacity    float64           `json:"gpu_capacity"`
	CPUUsed        float64           `json:"cpu_used"`
	MemoryUsed     float64           `json:"memory_used"`
	GPUUsed        float64           `json:"gpu_used"`
	NetworkLatency float64           `json:"network_latency"`
	WorkloadCount  int               `json:"workload_count"`
	Health         float64           `json:"health"`
	Location       string            `json:"location"`
	Labels         map[string]string `json:"labels"`
}

// ScoreBreakdown shows how a scheduling score was calculated.
type ScoreBreakdown struct {
	NodeID              string  `json:"node_id"`
	TotalScore          float64 `json:"total_score"`
	CPUScore            float64 `json:"cpu_score"`
	MemoryScore         float64 `json:"memory_score"`
	GPUScore            float64 `json:"gpu_score"`
	NetworkScore        float64 `json:"network_score"`
	HealthScore         float64 `json:"health_score"`
	AffinityScore       float64 `json:"affinity_score"`
	UtilizationPenalty  float64 `json:"utilization_penalty"`
	PriorityBonus       float64 `json:"priority_bonus"`
}

// SchedulingResult is the result of scheduling a workload.
type SchedulingResult struct {
	WorkloadID string         `json:"workload_id"`
	NodeID     string         `json:"node_id"`
	Score      float64        `json:"score"`
	Breakdown  ScoreBreakdown `json:"breakdown"`
	Timestamp  time.Time      `json:"timestamp"`
}

// Engine is the workload scheduler.
type Engine struct {
	mu    sync.RWMutex
	nodes map[string]*NodeInfo
	bus   *event.Bus
}

// NewEngine creates a new scheduler engine.
func NewEngine(bus *event.Bus) *Engine {
	return &Engine{
		nodes: make(map[string]*NodeInfo),
		bus:   bus,
	}
}

// RegisterNode adds or updates a node for scheduling.
func (e *Engine) RegisterNode(n *NodeInfo) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nodes[n.ID] = n
}

// UnregisterNode removes a node from scheduling.
func (e *Engine) UnregisterNode(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.nodes, id)
}

// Nodes returns all registered nodes.
func (e *Engine) Nodes() []*NodeInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*NodeInfo, 0, len(e.nodes))
	for _, n := range e.nodes {
		result = append(result, n)
	}
	return result
}

// Schedule finds the best node for a workload.
func (e *Engine) Schedule(w *Workload) *SchedulingResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	scores := e.scoreAll(w)
	if len(scores) == 0 {
		return nil
	}

	best := scores[0]
	result := &SchedulingResult{
		WorkloadID: w.ID,
		NodeID:     best.NodeID,
		Score:      best.TotalScore,
		Breakdown:  best,
		Timestamp:  time.Now(),
	}

	e.bus.Publish(event.NewEvent(event.JobAssigned, "scheduler", map[string]interface{}{
		"workload_id": w.ID,
		"node_id":     best.NodeID,
		"score":       best.TotalScore,
	}))

	return result
}

// ScheduleAll schedules multiple workloads across available nodes.
func (e *Engine) ScheduleAll(workloads []*Workload) []*SchedulingResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	var results []*SchedulingResult
	for _, w := range workloads {
		scores := e.scoreAllLocked(w)
		if len(scores) == 0 {
			continue
		}
		best := scores[0]
		results = append(results, &SchedulingResult{
			WorkloadID: w.ID,
			NodeID:     best.NodeID,
			Score:      best.TotalScore,
			Breakdown:  best,
			Timestamp:  time.Now(),
		})
		if node, ok := e.nodes[best.NodeID]; ok {
			node.CPUUsed += w.CPU
			node.MemoryUsed += w.Memory
			node.GPUUsed += w.GPU
			node.WorkloadCount++
		}
	}
	return results
}

func (e *Engine) scoreAll(w *Workload) []ScoreBreakdown {
	return e.scoreAllLocked(w)
}

func (e *Engine) scoreAllLocked(w *Workload) []ScoreBreakdown {
	var scores []ScoreBreakdown
	for _, node := range e.nodes {
		if node.Health <= 0 {
			continue
		}
		s := e.score(w, node)
		scores = append(scores, s)
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].TotalScore != scores[j].TotalScore {
			return scores[i].TotalScore > scores[j].TotalScore
		}
		return scores[i].NodeID < scores[j].NodeID
	})
	return scores
}

func (e *Engine) score(w *Workload, n *NodeInfo) ScoreBreakdown {
	s := ScoreBreakdown{NodeID: n.ID}

	cpuAvail := n.CPUCapacity - n.CPUUsed
	memAvail := n.MemoryCapacity - n.MemoryUsed
	gpuAvail := n.GPUCapacity - n.GPUUsed

	s.CPUScore = resourceScore(w.CPU, cpuAvail)
	s.MemoryScore = resourceScore(w.Memory, memAvail)
	s.GPUScore = resourceScore(w.GPU, gpuAvail)
	s.NetworkScore = networkScore(w.LatencyMax, n.NetworkLatency)
	s.HealthScore = n.Health

	s.AffinityScore = affinityScore(w, n)
	s.UtilizationPenalty = utilizationPenalty(n)
	s.PriorityBonus = float64(w.Priority) / 100.0

	s.TotalScore = s.CPUScore + s.MemoryScore + s.GPUScore +
		s.NetworkScore + s.HealthScore + s.AffinityScore -
		s.UtilizationPenalty + s.PriorityBonus

	return s
}

func resourceScore(need, available float64) float64 {
	if need <= 0 {
		return 1.0
	}
	if available <= 0 {
		return 0.0
	}
	if available < need {
		return available / need
	}
	// The node fits the requirement; reward surplus headroom so nodes with
	// more available capacity outscore nodes that barely fit.
	extra := (available - need) / available
	return 0.5 + 0.5*extra
}

func networkScore(maxLatency, actualLatency float64) float64 {
	if maxLatency <= 0 {
		return 1.0
	}
	if actualLatency <= 0 {
		return 1.0
	}
	if actualLatency > maxLatency {
		return 0.0
	}
	return 1.0 - (actualLatency / maxLatency)
}

func affinityScore(w *Workload, n *NodeInfo) float64 {
	score := 0.0
	count := 0
	for _, affinity := range w.Affinity {
		count++
		if n.Labels != nil {
			if v, ok := n.Labels[affinity]; ok && v == "true" {
				score += 1.0
			}
		}
	}
	if count > 0 {
		return score / float64(count)
	}

	for _, anti := range w.AntiAffinity {
		if n.Labels != nil {
			if v, ok := n.Labels[anti]; ok && v == "true" {
				return -1.0
			}
		}
	}
	return 0.0
}

func utilizationPenalty(n *NodeInfo) float64 {
	cpuUtil := 0.0
	if n.CPUCapacity > 0 {
		cpuUtil = n.CPUUsed / n.CPUCapacity
	}
	memUtil := 0.0
	if n.MemoryCapacity > 0 {
		memUtil = n.MemoryUsed / n.MemoryCapacity
	}
	avg := (cpuUtil + memUtil) / 2.0
	if avg > 0.8 {
		return (avg - 0.8) * 5.0
	}
	return 0.0
}

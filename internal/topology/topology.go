// Package topology provides the infrastructure topology engine.
// It uses the graph engine to model infrastructure as a graph and
// provides dependency analysis, criticality scoring, and failure propagation.
package topology

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/algorithms/graph"
	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/models"
)

// Engine is the topology engine that manages infrastructure graph.
type Engine struct {
	mu     sync.RWMutex
	g      *graph.Graph
	nodes  map[string]*models.Node
	edges  map[string]*models.Edge
	bus    *event.Bus
}

// NewEngine creates a new topology engine.
func NewEngine(bus *event.Bus) *Engine {
	return &Engine{
		g:     graph.New(true),
		nodes: make(map[string]*models.Node),
		edges: make(map[string]*models.Edge),
		bus:   bus,
	}
}

// AddNode adds an infrastructure node to the topology.
func (e *Engine) AddNode(n *models.Node) {
	e.mu.Lock()
	defer e.mu.Unlock()

	n.CreatedAt = time.Now()
	n.UpdatedAt = time.Now()
	e.nodes[n.ID] = n
	e.g.AddNode(n.ID, n.Name)

	e.bus.Publish(event.NewEvent(event.NodeDiscovered, "topology", map[string]interface{}{
		"node_id": n.ID,
		"type":    n.Type,
		"name":    n.Name,
	}))
}

// RemoveNode removes a node and its edges from the topology.
func (e *Engine) RemoveNode(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.nodes[id]; !ok {
		return
	}

	delete(e.nodes, id)
	e.g.RemoveNode(id)

	for edgeID, edge := range e.edges {
		if edge.Source == id || edge.Target == id {
			delete(e.edges, edgeID)
		}
	}

	e.bus.Publish(event.NewEvent(event.NodeRemoved, "topology", map[string]interface{}{
		"node_id": id,
	}))
}

// GetNode returns a node by ID.
func (e *Engine) GetNode(id string) (*models.Node, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	n, ok := e.nodes[id]
	return n, ok
}

// SetNodeHealth updates a node's health score (0-1) and returns whether it existed.
func (e *Engine) SetNodeHealth(id string, health float64) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	n, ok := e.nodes[id]
	if !ok {
		return false
	}
	n.Health = health
	n.UpdatedAt = time.Now()
	return true
}

// SetNodeState updates a node's lifecycle state and returns whether it existed.
func (e *Engine) SetNodeState(id string, state models.NodeState) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	n, ok := e.nodes[id]
	if !ok {
		return false
	}
	n.State = state
	n.UpdatedAt = time.Now()
	return true
}

// AddEdge adds a relationship between two nodes.
func (e *Engine) AddEdge(edge *models.Edge) {
	e.mu.Lock()
	defer e.mu.Unlock()

	edge.CreatedAt = time.Now()
	e.edges[edge.ID] = edge
	e.g.AddEdge(edge.Source, edge.Target, edge.Weight)
}

// RemoveEdge removes a relationship.
func (e *Engine) RemoveEdge(id string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.edges, id)
}

// Nodes returns all topology nodes.
func (e *Engine) Nodes() []*models.Node {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*models.Node, 0, len(e.nodes))
	for _, n := range e.nodes {
		result = append(result, n)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}

// Edges returns all topology edges.
func (e *Engine) Edges() []*models.Edge {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*models.Edge, 0, len(e.edges))
	for _, ed := range e.edges {
		result = append(result, ed)
	}
	return result
}

// Graph returns the underlying graph for algorithm operations.
func (e *Engine) Graph() *graph.Graph {
	return e.g
}

// DirectDependents returns nodes that directly depend on the given node.
func (e *Engine) DirectDependents(nodeID string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var dependents []string
	for _, edge := range e.edges {
		if edge.Target == nodeID && edge.Type == models.EdgeDependsOn {
			dependents = append(dependents, edge.Source)
		}
	}
	sort.Strings(dependents)
	return dependents
}

// IndirectDependents returns all nodes transitively dependent on the given node via BFS.
func (e *Engine) IndirectDependents(nodeID string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	visited := make(map[string]bool)
	var result []string

	// BFS backwards through depends_on edges
	var queue []string
	for _, edge := range e.edges {
		if edge.Target == nodeID && edge.Type == models.EdgeDependsOn {
			queue = append(queue, edge.Source)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		result = append(result, current)

		for _, edge := range e.edges {
			if edge.Target == current && edge.Type == models.EdgeDependsOn && !visited[edge.Source] {
				queue = append(queue, edge.Source)
			}
		}
	}

	sort.Strings(result)
	return result
}

// ImpactAnalysis calculates the impact of removing a node.
func (e *Engine) ImpactAnalysis(nodeID string) ImpactResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	direct := e.directDependents(nodeID)
	indirect := e.indirectDependents(nodeID)
	allAffected := len(direct) + len(indirect)

	totalNodes := len(e.nodes)
	criticality := 0.0
	if totalNodes > 1 {
		criticality = float64(allAffected) / float64(totalNodes-1)
	}

	return ImpactResult{
		NodeID:          nodeID,
		DirectDependents:  direct,
		IndirectDependents: indirect,
		AffectedCount:    allAffected,
		Criticality:      math.Min(criticality, 1.0),
	}
}

// ImpactResult holds the result of impact analysis.
type ImpactResult struct {
	NodeID           string   `json:"node_id"`
	DirectDependents  []string `json:"direct_dependents"`
	IndirectDependents []string `json:"indirect_dependents"`
	AffectedCount    int      `json:"affected_count"`
	Criticality      float64  `json:"criticality"`
}

// Bottlenecks returns nodes that are bottlenecks (high betweenness or single points of failure).
func (e *Engine) Bottlenecks() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	aps := e.g.ArticulationPoints()
	return aps
}

// SinglePointsOfFailure returns nodes whose removal would disconnect the graph.
func (e *Engine) SinglePointsOfFailure() []string {
	return e.Bottlenecks()
}

// RecoveryPath calculates shortest recovery path between two nodes.
func (e *Engine) RecoveryPath(from, to string) ([]string, float64) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := e.g.Dijkstra(from)
	path := result.ShortestPath(to)
	dist := result.Distances[to]
	return path, dist
}

// Cycles returns dependency cycles if any exist.
func (e *Engine) Cycles() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.g.DFSCyclic()
}

// TopologicalOrder returns nodes in dependency order.
func (e *Engine) TopologicalOrder() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.g.TopologicalSort()
}

// HealthScore calculates an overall topology health score (0-1).
func (e *Engine) HealthScore() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.nodes) == 0 {
		return 1.0
	}

	totalHealth := 0.0
	for _, n := range e.nodes {
		totalHealth += n.Health
	}
	return totalHealth / float64(len(e.nodes))
}

// directDependents is the internal version without locking.
func (e *Engine) directDependents(nodeID string) []string {
	var dependents []string
	for _, edge := range e.edges {
		if edge.Target == nodeID && edge.Type == models.EdgeDependsOn {
			dependents = append(dependents, edge.Source)
		}
	}
	sort.Strings(dependents)
	return dependents
}

// indirectDependents is the internal version without locking.
func (e *Engine) indirectDependents(nodeID string) []string {
	visited := make(map[string]bool)
	var result []string

	var queue []string
	for _, edge := range e.edges {
		if edge.Target == nodeID && edge.Type == models.EdgeDependsOn {
			queue = append(queue, edge.Source)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		result = append(result, current)

		for _, edge := range e.edges {
			if edge.Target == current && edge.Type == models.EdgeDependsOn && !visited[edge.Source] {
				queue = append(queue, edge.Source)
			}
		}
	}

	sort.Strings(result)
	return result
}

// TopologySnapshot is a serializable snapshot of the topology.
type TopologySnapshot struct {
	Nodes       []*models.Node `json:"nodes"`
	Edges       []*models.Edge `json:"edges"`
	HealthScore float64        `json:"health_score"`
	Timestamp   time.Time      `json:"timestamp"`
}

// Snapshot returns a serializable snapshot of the current topology.
func (e *Engine) Snapshot() TopologySnapshot {
	return TopologySnapshot{
		Nodes:       e.Nodes(),
		Edges:       e.Edges(),
		HealthScore: e.HealthScore(),
		Timestamp:   time.Now(),
	}
}

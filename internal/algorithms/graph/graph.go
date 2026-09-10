// Package graph implements a graph data structure with adjacency list representation
// and all major graph algorithms.
//
// Supported:
//   - Directed and undirected graphs
//   - Weighted and unweighted edges
//   - Dynamic node/edge addition and removal
//
// Algorithms:
//   - BFS: O(V + E)
//   - DFS: O(V + E)
//   - Dijkstra: O((V + E) log V)
//   - A*: O((V + E) log V)
//   - Topological sort: O(V + E)
//   - Cycle detection: O(V + E)
//   - Connected components: O(V + E)
//   - Strongly connected components (Tarjan): O(V + E)
//   - Articulation points: O(V + E)
//   - Bridges: O(V + E)
package graph

import (
	"container/heap"
	"math"
	"sort"
)

// Node represents a vertex in the graph.
type Node struct {
	ID       string
	Label    string
	Metadata map[string]interface{}
}

// Edge represents a directed or undirected edge.
type Edge struct {
	From     string
	To       string
	Weight   float64
	Metadata map[string]interface{}
}

// Graph is an adjacency list graph.
type Graph struct {
	directed bool
	nodes    map[string]*Node
	adj      map[string][]Edge
}

// New creates a new graph. directed=true for directed graph, false for undirected.
func New(directed bool) *Graph {
	return &Graph{
		directed: directed,
		nodes:    make(map[string]*Node),
		adj:      make(map[string][]Edge),
	}
}

// AddNode adds a node to the graph. O(1).
func (g *Graph) AddNode(id, label string) {
	if _, exists := g.nodes[id]; exists {
		return
	}
	g.nodes[id] = &Node{ID: id, Label: label, Metadata: make(map[string]interface{})}
	if g.adj[id] == nil {
		g.adj[id] = make([]Edge, 0)
	}
}

// AddEdge adds an edge between two nodes. O(1).
func (g *Graph) AddEdge(from, to string, weight float64) {
	g.adj[from] = append(g.adj[from], Edge{From: from, To: to, Weight: weight})
	if !g.directed {
		g.adj[to] = append(g.adj[to], Edge{From: to, To: from, Weight: weight})
	}
}

// RemoveNode removes a node and all its edges. O(V + E).
func (g *Graph) RemoveNode(id string) {
	delete(g.nodes, id)
	delete(g.adj, id)
	for nodeID, edges := range g.adj {
		filtered := edges[:0]
		for _, e := range edges {
			if e.To != id {
				filtered = append(filtered, e)
			}
		}
		g.adj[nodeID] = filtered
	}
}

// Nodes returns all nodes. O(V).
func (g *Graph) Nodes() []*Node {
	result := make([]*Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		result = append(result, n)
	}
	return result
}

// Node returns a node by ID. O(1).
func (g *Graph) Node(id string) (*Node, bool) {
	n, ok := g.nodes[id]
	return n, ok
}

// Edges returns all edges from a node. O(degree).
func (g *Graph) Edges(id string) []Edge {
	return g.adj[id]
}

// Neighbors returns the IDs of adjacent nodes. O(degree).
func (g *Graph) Neighbors(id string) []string {
	neighbors := make([]string, 0, len(g.adj[id]))
	for _, e := range g.adj[id] {
		neighbors = append(neighbors, e.To)
	}
	return neighbors
}

// NodeCount returns the number of nodes. O(1).
func (g *Graph) NodeCount() int {
	return len(g.nodes)
}

// EdgeCount returns the total number of edges. O(V).
func (g *Graph) EdgeCount() int {
	count := 0
	for _, edges := range g.adj {
		count += len(edges)
	}
	if !g.directed {
		return count / 2
	}
	return count
}

// IsDirected returns whether the graph is directed.
func (g *Graph) IsDirected() bool {
	return g.directed
}

// HasNode checks if a node exists. O(1).
func (g *Graph) HasNode(id string) bool {
	_, ok := g.nodes[id]
	return ok
}

// HasEdge checks if an edge exists from->to. O(degree).
func (g *Graph) HasEdge(from, to string) bool {
	for _, e := range g.adj[from] {
		if e.To == to {
			return true
		}
	}
	return false
}

// GetEdge returns the edge from->to if it exists.
func (g *Graph) GetEdge(from, to string) (Edge, bool) {
	for _, e := range g.adj[from] {
		if e.To == to {
			return e, true
		}
	}
	return Edge{}, false
}

// Clone creates a deep copy of the graph.
func (g *Graph) Clone() *Graph {
	ng := New(g.directed)
	for id, n := range g.nodes {
		ng.nodes[id] = &Node{ID: n.ID, Label: n.Label}
		ng.adj[id] = make([]Edge, len(g.adj[id]))
		copy(ng.adj[id], g.adj[id])
	}
	return ng
}

// BFS performs breadth-first search starting from the given node.
// Returns the nodes in visit order.
// O(V + E).
func (g *Graph) BFS(start string) []string {
	visited := make(map[string]bool)
	queue := []string{start}
	visited[start] = true
	var order []string

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for _, edge := range g.adj[node] {
			if !visited[edge.To] {
				visited[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}
	return order
}

// BFSWithDistance performs BFS and returns shortest distances (in hops).
func (g *Graph) BFSWithDistance(start string) map[string]int {
	dist := make(map[string]int)
	for id := range g.nodes {
		dist[id] = math.MaxInt64
	}
	dist[start] = 0
	queue := []string{start}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, edge := range g.adj[node] {
			if dist[edge.To] == math.MaxInt64 {
				dist[edge.To] = dist[node] + 1
				queue = append(queue, edge.To)
			}
		}
	}
	return dist
}

// DFS performs depth-first search (iterative) starting from the given node.
// Returns the nodes in visit order.
// O(V + E).
func (g *Graph) DFS(start string) []string {
	visited := make(map[string]bool)
	stack := []string{start}
	var order []string

	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[node] {
			continue
		}
		visited[node] = true
		order = append(order, node)

		for i := len(g.adj[node]) - 1; i >= 0; i-- {
			edge := g.adj[node][i]
			if !visited[edge.To] {
				stack = append(stack, edge.To)
			}
		}
	}
	return order
}

// DFSCyclic checks if the graph has a cycle using DFS.
func (g *Graph) DFSCyclic() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for id := range g.nodes {
		if !visited[id] {
			if g.dfsCycleUtil(id, visited, recStack) {
				return true
			}
		}
	}
	return false
}

func (g *Graph) dfsCycleUtil(node string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, edge := range g.adj[node] {
		if !visited[edge.To] {
			if g.dfsCycleUtil(edge.To, visited, recStack) {
				return true
			}
		} else if recStack[edge.To] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// TopologicalSort performs topological sort using Kahn's algorithm.
// Returns an ordering, or nil if the graph has a cycle.
// O(V + E).
func (g *Graph) TopologicalSort() []string {
	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = 0
	}
	for _, edges := range g.adj {
		for _, e := range edges {
			inDegree[e.To]++
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		for _, edge := range g.adj[node] {
			inDegree[edge.To]--
			if inDegree[edge.To] == 0 {
				queue = append(queue, edge.To)
			}
		}
	}

	if len(sorted) != len(g.nodes) {
		return nil
	}
	return sorted
}

// ConnectedComponents returns the connected components (undirected graph).
// O(V + E).
func (g *Graph) ConnectedComponents() [][]string {
	visited := make(map[string]bool)
	var components [][]string

	for id := range g.nodes {
		if !visited[id] {
			var component []string
			stack := []string{id}
			for len(stack) > 0 {
				node := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if visited[node] {
					continue
				}
				visited[node] = true
				component = append(component, node)
				for _, edge := range g.adj[node] {
					if !visited[edge.To] {
						stack = append(stack, edge.To)
					}
				}
			}
			components = append(components, component)
		}
	}
	return components
}

// TarjanSCC finds strongly connected components using Tarjan's algorithm.
// O(V + E).
func (g *Graph) TarjanSCC() [][]string {
	index := 0
	var stack []string
	indices := make(map[string]int)
	lowlink := make(map[string]int)
	onStack := make(map[string]bool)
	var sccs [][]string

	var strongConnect func(v string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, edge := range g.adj[v] {
			w := edge.To
			if _, ok := indices[w]; !ok {
				strongConnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}

		if lowlink[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	for id := range g.nodes {
		if _, ok := indices[id]; !ok {
			strongConnect(id)
		}
	}
	return sccs
}

// ArticulationPoints finds all articulation points using Tarjan's algorithm.
// Connectivity is treated as undirected (edge direction is ignored), so the
// result is well-defined for both directed and undirected graph instances.
// O(V + E).
func (g *Graph) ArticulationPoints() []string {
	undirected := g.undirectedNeighbors()
	index := 0
	indices := make(map[string]int)
	lowlink := make(map[string]int)
	visited := make(map[string]bool)
	ap := make(map[string]bool)

	var dfs func(u, parent string)
	dfs = func(u, parent string) {
		visited[u] = true
		indices[u] = index
		lowlink[u] = index
		index++
		children := 0

		for _, v := range undirected[u] {
			if !visited[v] {
				children++
				dfs(v, u)
				if lowlink[v] < lowlink[u] {
					lowlink[u] = lowlink[v]
				}
				if parent != "" && lowlink[v] >= indices[u] {
					ap[u] = true
				}
			} else if v != parent {
				if indices[v] < lowlink[u] {
					lowlink[u] = indices[v]
				}
			}
		}

		if parent == "" && children > 1 {
			ap[u] = true
		}
	}

	for id := range g.nodes {
		if !visited[id] {
			dfs(id, "")
		}
	}

	var result []string
	for id := range ap {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// Bridges finds all bridges using Tarjan's algorithm. Edge direction is
// ignored so the result is well-defined for directed graph instances.
// O(V + E).
func (g *Graph) Bridges() [][2]string {
	undirected := g.undirectedNeighbors()
	index := 0
	indices := make(map[string]int)
	lowlink := make(map[string]int)
	visited := make(map[string]bool)
	bridges := make(map[[2]string]bool)

	var dfs func(u, parent string)
	dfs = func(u, parent string) {
		visited[u] = true
		indices[u] = index
		lowlink[u] = index
		index++

		for _, v := range undirected[u] {
			if !visited[v] {
				dfs(v, u)
				if lowlink[v] < lowlink[u] {
					lowlink[u] = lowlink[v]
				}
				if lowlink[v] > indices[u] {
					bridges[[2]string{u, v}] = true
				}
			} else if v != parent {
				if indices[v] < lowlink[u] {
					lowlink[u] = indices[v]
				}
			}
		}
	}

	for id := range g.nodes {
		if !visited[id] {
			dfs(id, "")
		}
	}

	result := make([][2]string, 0, len(bridges))
	for b := range bridges {
		result = append(result, [2]string{b[0], b[1]})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i][0] != result[j][0] {
			return result[i][0] < result[j][0]
		}
		return result[i][1] < result[j][1]
	})
	return result
}

// undirectedNeighbors builds a symmetric adjacency map treating every edge
// as bidirectional, regardless of the graph's directed flag.
func (g *Graph) undirectedNeighbors() map[string][]string {
	result := make(map[string][]string, len(g.nodes))
	seen := make(map[[2]string]bool)
	for u, edges := range g.adj {
		for _, e := range edges {
			pair := [2]string{u, e.To}
			if seen[pair] {
				continue
			}
			seen[pair] = true
			result[u] = append(result[u], e.To)
			result[e.To] = append(result[e.To], u)
		}
	}
	for id := range g.nodes {
		if _, ok := result[id]; !ok {
			result[id] = nil
		}
	}
	return result
}

// dijkstraNode is used internally for Dijkstra's algorithm.
type dijkstraNode struct {
	id       string
	distance float64
	index    int
}

type dijkstraHeap []*dijkstraNode

func (h dijkstraHeap) Len() int            { return len(h) }
func (h dijkstraHeap) Less(i, j int) bool  { return h[i].distance < h[j].distance }
func (h dijkstraHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *dijkstraHeap) Push(x interface{}) { n := x.(*dijkstraNode); n.index = len(*h); *h = append(*h, n) }
func (h *dijkstraHeap) Pop() interface{} {
	old := *h
	n := old[len(old)-1]
	old[len(old)-1] = nil
	n.index = -1
	*h = old[:len(old)-1]
	return n
}

// DijkstraResult holds the result of Dijkstra's algorithm.
type DijkstraResult struct {
	Distances map[string]float64
	Previous  map[string]string
	Path      []string
}

// Dijkstra computes shortest paths from a source node to all reachable nodes.
// O((V + E) log V).
func (g *Graph) Dijkstra(source string) DijkstraResult {
	dist := make(map[string]float64)
	prev := make(map[string]string)
	for id := range g.nodes {
		dist[id] = math.Inf(1)
	}
	dist[source] = 0

	h := &dijkstraHeap{}
	heap.Push(h, &dijkstraNode{id: source, distance: 0})

	for h.Len() > 0 {
		u := heap.Pop(h).(*dijkstraNode)
		if u.distance > dist[u.id] {
			continue
		}
		for _, edge := range g.adj[u.id] {
			alt := dist[u.id] + edge.Weight
			if alt < dist[edge.To] {
				dist[edge.To] = alt
				prev[edge.To] = u.id
				heap.Push(h, &dijkstraNode{id: edge.To, distance: alt})
			}
		}
	}

	return DijkstraResult{
		Distances: dist,
		Previous:  prev,
	}
}

// ShortestPath reconstructs the shortest path from Dijkstra result.
func (r DijkstraResult) ShortestPath(target string) []string {
	if _, ok := r.Distances[target]; !ok || math.IsInf(r.Distances[target], 1) {
		return nil
	}
	var path []string
	for at := target; at != ""; at = r.Previous[at] {
		path = append([]string{at}, path...)
	}
	return path
}

// AStar performs A* search from start to goal.
// heuristic is a function that estimates the cost from a node to the goal.
// Returns the path and total cost, or nil path if no path exists.
// O((V + E) log V).
func (g *Graph) AStar(start, goal string, heuristic func(from, to string) float64) ([]string, float64) {
	gScore := make(map[string]float64)
	fScore := make(map[string]float64)
	cameFrom := make(map[string]string)
	closedSet := make(map[string]bool)

	for id := range g.nodes {
		gScore[id] = math.Inf(1)
		fScore[id] = math.Inf(1)
	}
	gScore[start] = 0
	fScore[start] = heuristic(start, goal)

	h := &dijkstraHeap{}
	heap.Push(h, &dijkstraNode{id: start, distance: fScore[start]})

	for h.Len() > 0 {
		current := heap.Pop(h).(*dijkstraNode).id

		if current == goal {
			var path []string
			for at := goal; at != ""; at = cameFrom[at] {
				path = append([]string{at}, path...)
			}
			return path, gScore[goal]
		}

		closedSet[current] = true

		for _, edge := range g.adj[current] {
			if closedSet[edge.To] {
				continue
			}
			tentativeG := gScore[current] + edge.Weight
			if tentativeG < gScore[edge.To] {
				cameFrom[edge.To] = current
				gScore[edge.To] = tentativeG
				fScore[edge.To] = tentativeG + heuristic(edge.To, goal)
				heap.Push(h, &dijkstraNode{id: edge.To, distance: fScore[edge.To]})
			}
		}
	}

	return nil, math.Inf(1)
}

// DirectedGraph returns true if the graph is directed.
func (g *Graph) DirectedGraph() bool {
	return g.directed
}

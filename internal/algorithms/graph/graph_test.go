package graph

import (
	"math"
	"testing"
)

func TestAddNode(t *testing.T) {
	g := New(false)
	g.AddNode("A", "Alpha")
	g.AddNode("B", "Beta")

	if g.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", g.NodeCount())
	}
	if !g.HasNode("A") {
		t.Error("expected node A")
	}
}

func TestAddEdge(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddEdge("A", "B", 1.0)

	if !g.HasEdge("A", "B") {
		t.Error("expected edge A->B")
	}
	if g.HasEdge("B", "A") {
		t.Error("directed: expected no edge B->A")
	}
}

func TestAddEdgeUndirected(t *testing.T) {
	g := New(false)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddEdge("A", "B", 1.0)

	if !g.HasEdge("A", "B") {
		t.Error("expected edge A->B")
	}
	if !g.HasEdge("B", "A") {
		t.Error("undirected: expected edge B->A")
	}
}

func TestRemoveNode(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "C", 1.0)

	g.RemoveNode("B")
	if g.HasNode("B") {
		t.Error("expected B removed")
	}
	if g.HasEdge("A", "B") {
		t.Error("expected A->B removed")
	}
}

func TestNeighbors(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("A", "C", 1.0)

	n := g.Neighbors("A")
	if len(n) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(n))
	}
}

func TestBFS(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("A", "C", 1.0)
	g.AddEdge("B", "D", 1.0)
	g.AddEdge("C", "D", 1.0)

	order := g.BFS("A")
	if len(order) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(order))
	}
	if order[0] != "A" {
		t.Errorf("expected first node A, got %s", order[0])
	}
}

func TestBFSDistance(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("A", "C", 1.0)
	g.AddEdge("B", "D", 1.0)

	dist := g.BFSWithDistance("A")
	if dist["A"] != 0 {
		t.Errorf("expected dist A=0, got %d", dist["A"])
	}
	if dist["D"] != 2 {
		t.Errorf("expected dist D=2, got %d", dist["D"])
	}
}

func TestDFS(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("A", "C", 1.0)

	order := g.DFS("A")
	if len(order) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(order))
	}
	if order[0] != "A" {
		t.Errorf("expected first node A, got %s", order[0])
	}
}

func TestDFSCycleDetection(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "C", 1.0)
	g.AddEdge("C", "A", 1.0)

	if !g.DFSCyclic() {
		t.Error("expected cycle detected")
	}
}

func TestDFSNoCycle(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "C", 1.0)

	if g.DFSCyclic() {
		t.Error("expected no cycle")
	}
}

func TestTopologicalSort(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("A", "C", 1.0)
	g.AddEdge("B", "D", 1.0)
	g.AddEdge("C", "D", 1.0)

	order := g.TopologicalSort()
	if order == nil {
		t.Fatal("expected valid topological order")
	}
	if len(order) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(order))
	}
	idx := make(map[string]int)
	for i, id := range order {
		idx[id] = i
	}
	if idx["A"] > idx["B"] {
		t.Error("A should come before B")
	}
	if idx["A"] > idx["C"] {
		t.Error("A should come before C")
	}
}

func TestTopologicalSortCycle(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "A", 1.0)

	if g.TopologicalSort() != nil {
		t.Error("expected nil for cyclic graph")
	}
}

func TestConnectedComponents(t *testing.T) {
	g := New(false)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("C", "D", 1.0)

	cc := g.ConnectedComponents()
	if len(cc) != 2 {
		t.Errorf("expected 2 components, got %d", len(cc))
	}
}

func TestTarjanSCC(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "C", 1.0)
	g.AddEdge("C", "A", 1.0)
	g.AddEdge("B", "D", 1.0)

	sccs := g.TarjanSCC()
	if len(sccs) != 2 {
		t.Errorf("expected 2 SCCs, got %d", len(sccs))
	}
}

func TestArticulationPoints(t *testing.T) {
	g := New(false)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "C", 1.0)
	g.AddEdge("B", "D", 1.0)

	aps := g.ArticulationPoints()
	if len(aps) != 1 || aps[0] != "B" {
		t.Errorf("expected [B], got %v", aps)
	}
}

func TestBridges(t *testing.T) {
	g := New(false)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "C", 1.0)

	bridges := g.Bridges()
	if len(bridges) != 2 {
		t.Errorf("expected 2 bridges, got %d", len(bridges))
	}
}

func TestDijkstra(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("A", "C", 4.0)
	g.AddEdge("B", "C", 2.0)
	g.AddEdge("B", "D", 6.0)
	g.AddEdge("C", "D", 3.0)

	result := g.Dijkstra("A")
	if result.Distances["D"] != 6.0 {
		t.Errorf("expected dist D=6, got %f", result.Distances["D"])
	}
	path := result.ShortestPath("D")
	if len(path) == 0 {
		t.Error("expected non-empty path")
	}
}

func TestDijkstraNoPath(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")

	result := g.Dijkstra("A")
	if !math.IsInf(result.Distances["B"], 1) {
		t.Error("expected infinity for unreachable node")
	}
}

func TestAStar(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddNode("D", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("A", "C", 4.0)
	g.AddEdge("B", "C", 2.0)
	g.AddEdge("B", "D", 6.0)
	g.AddEdge("C", "D", 3.0)

	heuristic := func(from, to string) float64 { return 0 }
	path, cost := g.AStar("A", "D", heuristic)
	if path == nil {
		t.Fatal("expected path")
	}
	if cost != 6.0 {
		t.Errorf("expected cost 6, got %f", cost)
	}
}

func TestAStarNoPath(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")

	heuristic := func(from, to string) float64 { return 0 }
	path, _ := g.AStar("A", "B", heuristic)
	if path != nil {
		t.Error("expected nil path")
	}
}

func TestEdgeCount(t *testing.T) {
	g := New(false)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("C", "")
	g.AddEdge("A", "B", 1.0)
	g.AddEdge("B", "C", 1.0)

	if g.EdgeCount() != 2 {
		t.Errorf("expected 2 edges, got %d", g.EdgeCount())
	}
}

func TestDirectedEdgeCount(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddEdge("A", "B", 1.0)

	if g.EdgeCount() != 1 {
		t.Errorf("expected 1 edge, got %d", g.EdgeCount())
	}
}

func TestClone(t *testing.T) {
	g := New(true)
	g.AddNode("A", "Alpha")
	g.AddEdge("A", "B", 1.0)

	clone := g.Clone()
	clone.AddNode("C", "")
	clone.AddEdge("A", "C", 2.0)

	if g.HasNode("C") {
		t.Error("original should not have C")
	}
	if !clone.HasNode("C") {
		t.Error("clone should have C")
	}
}

func TestGetEdge(t *testing.T) {
	g := New(true)
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddEdge("A", "B", 42.0)

	e, ok := g.GetEdge("A", "B")
	if !ok || e.Weight != 42.0 {
		t.Errorf("expected edge with weight 42, got %v, %v", e, ok)
	}
}

func TestDijkstraPathReconstruction(t *testing.T) {
	g := New(true)
	g.AddNode("S", "")
	g.AddNode("A", "")
	g.AddNode("B", "")
	g.AddNode("T", "")
	g.AddEdge("S", "A", 1.0)
	g.AddEdge("S", "B", 5.0)
	g.AddEdge("A", "T", 2.0)
	g.AddEdge("B", "T", 1.0)

	result := g.Dijkstra("S")
	path := result.ShortestPath("T")
	if path == nil || len(path) != 3 {
		t.Errorf("expected path [S, A, T], got %v", path)
	}
	if path[0] != "S" || path[1] != "A" || path[2] != "T" {
		t.Errorf("expected [S A T], got %v", path)
	}
}

func BenchmarkBFS(b *testing.B) {
	g := New(true)
	for i := 0; i < 10000; i++ {
		g.AddNode(string(rune('A'+i%26))+string(rune('0'+i/26)), "")
		if i > 0 {
			g.AddEdge(string(rune('A'+(i-1)%26))+string(rune('0'+(i-1)/26)),
				string(rune('A'+i%26))+string(rune('0'+i/26)), 1.0)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.BFS("A0")
	}
}

func BenchmarkDFS(b *testing.B) {
	g := New(true)
	for i := 0; i < 10000; i++ {
		g.AddNode(string(rune('A'+i%26))+string(rune('0'+i/26)), "")
		if i > 0 {
			g.AddEdge(string(rune('A'+(i-1)%26))+string(rune('0'+(i-1)/26)),
				string(rune('A'+i%26))+string(rune('0'+i/26)), 1.0)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.DFS("A0")
	}
}

func BenchmarkDijkstra(b *testing.B) {
	g := New(true)
	for i := 0; i < 1000; i++ {
		g.AddNode(string(rune('A'+i%26))+string(rune('0'+i/26)), "")
		if i > 0 {
			g.AddEdge(string(rune('A'+(i-1)%26))+string(rune('0'+(i-1)/26)),
				string(rune('A'+i%26))+string(rune('0'+i/26)), float64(i))
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Dijkstra("A0")
	}
}

func BenchmarkTopologicalSort(b *testing.B) {
	g := New(true)
	for i := 0; i < 1000; i++ {
		g.AddNode(string(rune('A'+i%26))+string(rune('0'+i/26)), "")
		if i > 0 {
			g.AddEdge(string(rune('A'+(i-1)%26))+string(rune('0'+(i-1)/26)),
				string(rune('A'+i%26))+string(rune('0'+i/26)), 1.0)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.TopologicalSort()
	}
}

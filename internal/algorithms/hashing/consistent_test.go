package hashing

import (
	"fmt"
	"math"
	"testing"
)

func TestAddNode(t *testing.T) {
	ring := NewHashRing(50)
	ring.AddNode("node-1")
	ring.AddNode("node-2")

	if ring.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", ring.NodeCount())
	}
}

func TestGetNodeConsistent(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("node-1")
	ring.AddNode("node-2")
	ring.AddNode("node-3")

	// Same key should always map to the same node
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		n1 := ring.GetNode(key)
		n2 := ring.GetNode(key)
		if n1 != n2 {
			t.Fatalf("key %s mapped to different nodes: %s vs %s", key, n1, n2)
		}
	}
}

func TestGetNodeEmpty(t *testing.T) {
	ring := NewHashRing(100)
	if n := ring.GetNode("key"); n != "" {
		t.Errorf("expected empty node for empty ring, got %s", n)
	}
}

func TestGetNodesReplication(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("a")
	ring.AddNode("b")
	ring.AddNode("c")

	nodes := ring.GetNodes("some-key", 2)
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0] == nodes[1] {
		t.Error("expected distinct nodes")
	}
}

func TestRemoveNode(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("a")
	ring.AddNode("b")
	ring.AddNode("c")

	key := "test-key"
	nodeBefore := ring.GetNode(key)

	ring.RemoveNode(nodeBefore)

	after := ring.GetNode(key)
	if after == "" {
		t.Error("expected another node to take over")
	}
	if after == nodeBefore {
		t.Error("removed node should not be returned")
	}
}

func TestMinimalRemapOnAdd(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("a")
	ring.AddNode("b")
	ring.AddNode("c")

	numKeys := 10000
	keys := make([]string, numKeys)
	before := make(map[string]string)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		keys[i] = key
		before[key] = ring.GetNode(key)
	}

	ring.AddNode("d")

	remapped := 0
	for _, key := range keys {
		if ring.GetNode(key) != before[key] {
			remapped++
		}
	}

	remapRatio := float64(remapped) / float64(numKeys)
	expected := 1.0 / 4.0
	if math.Abs(remapRatio-expected) > 0.1 {
		t.Errorf("expected ~25%% remap, got %.1f%%", remapRatio*100)
	}
}

func TestMinimalRemapOnRemove(t *testing.T) {
	ring := NewHashRing(150)
	ring.AddNode("a")
	ring.AddNode("b")
	ring.AddNode("c")

	numKeys := 10000
	keys := make([]string, numKeys)
	before := make(map[string]string)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		keys[i] = key
		before[key] = ring.GetNode(key)
	}

	ring.RemoveNode("a")

	remapped := 0
	for _, key := range keys {
		if ring.GetNode(key) != before[key] {
			remapped++
		}
	}

	remapRatio := float64(remapped) / float64(numKeys)
	expected := 1.0 / 3.0
	if math.Abs(remapRatio-expected) > 0.1 {
		t.Errorf("expected ~33%% remap, got %.1f%%", remapRatio*100)
	}
}

func TestDistributionBalanced(t *testing.T) {
	ring := NewHashRing(200)
	ring.AddNode("a")
	ring.AddNode("b")
	ring.AddNode("c")
	ring.AddNode("d")

	numKeys := 10000
	keys := make([]string, numKeys)
	for i := 0; i < numKeys; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	dist := ring.Distribution(keys)
	if len(dist) != 4 {
		t.Fatalf("expected 4 nodes in distribution, got %d", len(dist))
	}

	// Check reasonable balance (each node should have roughly 25%)
	for node, count := range dist {
		ratio := float64(count) / float64(numKeys)
		if ratio < 0.15 || ratio > 0.35 {
			t.Errorf("node %s distribution %.1f%% outside expected range 15-35%%", node, ratio*100)
		}
	}
}

func TestNodes(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("b")
	ring.AddNode("a")

	nodes := ring.Nodes()
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0] != "a" || nodes[1] != "b" {
		t.Errorf("expected sorted [a b], got %v", nodes)
	}
}

func TestRingNotEmpty(t *testing.T) {
	ring := NewHashRing(100)
	ring.AddNode("node")
	if len(ring.Ring()) == 0 {
		t.Error("expected non-empty ring")
	}
}

func BenchmarkConsistentGet(b *testing.B) {
	ring := NewHashRing(150)
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		ring.AddNode(n)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ring.GetNode(fmt.Sprintf("key-%d", i))
	}
}

func BenchmarkConsistentAdd(b *testing.B) {
	ring := NewHashRing(150)
	for i := 0; i < 10; i++ {
		ring.AddNode(fmt.Sprintf("node-%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ring.AddNode(fmt.Sprintf("new-node-%d", i))
		ring.RemoveNode(fmt.Sprintf("new-node-%d", i))
	}
}

func BenchmarkModuloHashAdd(b *testing.B) {
	nodes := []string{"a", "b", "c", "d", "e"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		n := nodes[i%len(nodes)]
		_ = n
	}
}
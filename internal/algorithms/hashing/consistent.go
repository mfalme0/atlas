// Package hashing provides consistent hashing with virtual nodes.
//
// Consistent hashing distributes keys across nodes such that minimal keys
// are remapped when a node joins or leaves. Virtual nodes improve distribution.
//
// Operations:
//
//	AddNode:    O(v * log n) where v = virtual nodes per physical node
//	RemoveNode: O(v * log n)
//	Lookup:     O(log n)
package hashing

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
)

// HashRing is a consistent hash ring with virtual nodes.
type HashRing struct {
	mu           sync.RWMutex
	ring         []uint32
	nodes        map[uint32]string
	vnodes       int
	nodeMap      map[string]bool
}

// NewHashRing creates a new consistent hash ring.
func NewHashRing(virtualNodes int) *HashRing {
	if virtualNodes <= 0 {
		virtualNodes = 150
	}
	return &HashRing{
		ring:       make([]uint32, 0),
		nodes:      make(map[uint32]string),
		vnodes:     virtualNodes,
		nodeMap:    make(map[string]bool),
	}
}

// AddNode adds a node to the ring with virtual nodes. O(v * log n).
func (h *HashRing) AddNode(node string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.nodeMap[node] {
		return
	}
	h.nodeMap[node] = true

	for i := 0; i < h.vnodes; i++ {
		key := fmt.Sprintf("%s#%d", node, i)
		hash := crc32.ChecksumIEEE([]byte(key))
		h.ring = append(h.ring, hash)
		h.nodes[hash] = node
	}
	sort.Slice(h.ring, func(i, j int) bool {
		return h.ring[i] < h.ring[j]
	})
}

// RemoveNode removes a node and its virtual nodes from the ring. O(v * log n).
func (h *HashRing) RemoveNode(node string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.nodeMap[node] {
		return
	}
	delete(h.nodeMap, node)

	newRing := h.ring[:0]
	for _, hash := range h.ring {
		if h.nodes[hash] != node {
			newRing = append(newRing, hash)
		} else {
			delete(h.nodes, hash)
		}
	}
	h.ring = newRing
}

// GetNode returns the node responsible for the given key. O(log n).
func (h *HashRing) GetNode(key string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 {
		return ""
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hash
	})

	if idx >= len(h.ring) {
		idx = 0
	}

	return h.nodes[h.ring[idx]]
}

// GetNodes returns multiple nodes for replication. O(log n).
func (h *HashRing) GetNodes(key string, count int) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 {
		return nil
	}

	hash := crc32.ChecksumIEEE([]byte(key))
	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hash
	})

	seen := make(map[string]bool)
	var result []string

	for i := 0; len(result) < count && i < len(h.ring); i++ {
		pos := (idx + i) % len(h.ring)
		node := h.nodes[h.ring[pos]]
		if !seen[node] {
			seen[node] = true
			result = append(result, node)
		}
	}

	return result
}

// NodeCount returns the number of physical nodes. O(1).
func (h *HashRing) NodeCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.nodeMap)
}

// Nodes returns all physical nodes. O(n).
func (h *HashRing) Nodes() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]string, 0, len(h.nodeMap))
	for node := range h.nodeMap {
		result = append(result, node)
	}
	sort.Strings(result)
	return result
}

// Distribution returns the key distribution across nodes. O(k).
func (h *HashRing) Distribution(keys []string) map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	dist := make(map[string]int)
	for node := range h.nodeMap {
		dist[node] = 0
	}
	for _, key := range keys {
		node := h.getNodeLocked(key)
		if node != "" {
			dist[node]++
		}
	}
	return dist
}

// RemapCount returns how many keys would move if a node is added/removed. O(k).
func (h *HashRing) RemapCount(keys []string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, key := range keys {
		_ = h.getNodeLocked(key)
		count++
	}
	return count
}

func (h *HashRing) getNodeLocked(key string) string {
	if len(h.ring) == 0 {
		return ""
	}
	hash := crc32.ChecksumIEEE([]byte(key))
	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hash
	})
	if idx >= len(h.ring) {
		idx = 0
	}
	return h.nodes[h.ring[idx]]
}

// Ring returns the sorted ring hashes for visualization.
func (h *HashRing) Ring() []uint32 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]uint32, len(h.ring))
	copy(result, h.ring)
	return result
}

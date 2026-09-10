// Package hashtable provides a chaining hash map keyed by comparable keys.
//
// Collisions are resolved with singly-linked buckets; the table doubles in
// size when the load factor exceeds 0.75 and halves when it drops below 0.2.
// Methods are NOT safe for concurrent use; guard externally when shared.
package hashtable

import (
	"fmt"
	"math"
)

type bucketNode[K comparable, V any] struct {
	key   K
	value V
	next  *bucketNode[K, V]
}

// HashMap is a hash map with chaining and dynamic resizing.
type HashMap[K comparable, V any] struct {
	buckets []*bucketNode[K, V]
	size    int
	load    float64
}

// DefaultCapacity is the initial number of buckets.
const DefaultCapacity = 16

// New creates an empty hash map with the default capacity.
func New[K comparable, V any]() *HashMap[K, V] {
	return &HashMap[K, V]{
		buckets: make([]*bucketNode[K, V], DefaultCapacity),
		load:    0.75,
	}
}

// NewWithCapacity creates a hash map with at least the given capacity.
func NewWithCapacity[K comparable, V any](capacity int) *HashMap[K, V] {
	if capacity < 1 {
		capacity = DefaultCapacity
	}
	// Round up to the next power of two for stable hashing.
	for capacity < DefaultCapacity {
		capacity *= 2
	}
	return &HashMap[K, V]{
		buckets: make([]*bucketNode[K, V], capacity),
		load:    0.75,
	}
}

// Size returns the number of entries.
func (m *HashMap[K, V]) Size() int {
	return m.size
}

// IsEmpty reports whether the map has no entries.
func (m *HashMap[K, V]) IsEmpty() bool {
	return m.size == 0
}

// Capacity returns the current bucket count.
func (m *HashMap[K, V]) Capacity() int {
	return len(m.buckets)
}

// Put inserts or updates a key. Returns the previous value if present.
func (m *HashMap[K, V]) Put(key K, value V) (V, bool) {
	idx := m.index(key)
	prev, ok := m.scan(idx, key)
	if ok {
		old := prev.value
		prev.value = value
		return old, true
	}
	m.buckets[idx] = &bucketNode[K, V]{key: key, value: value, next: m.buckets[idx]}
	m.size++
	if float64(m.size)/float64(len(m.buckets)) > m.load {
		m.resize(len(m.buckets) * 2)
	}
	var zero V
	return zero, false
}

// Get retrieves a value by key.
func (m *HashMap[K, V]) Get(key K) (V, bool) {
	idx := m.index(key)
	for n := m.buckets[idx]; n != nil; n = n.next {
		if n.key == key {
			return n.value, true
		}
	}
	var zero V
	return zero, false
}

// Delete removes a key and returns the removed value.
func (m *HashMap[K, V]) Delete(key K) (V, bool) {
	idx := m.index(key)
	head := m.buckets[idx]
	if head == nil {
		var zero V
		return zero, false
	}
	if head.key == key {
		m.buckets[idx] = head.next
		m.size--
		m.maybeShrink()
		return head.value, true
	}
	for prev := head; prev.next != nil; prev = prev.next {
		if prev.next.key == key {
			node := prev.next
			prev.next = node.next
			m.size--
			m.maybeShrink()
			return node.value, true
		}
	}
	var zero V
	return zero, false
}

// Contains reports whether a key exists.
func (m *HashMap[K, V]) Contains(key K) bool {
	_, ok := m.Get(key)
	return ok
}

// Keys returns all keys in unspecified order.
func (m *HashMap[K, V]) Keys() []K {
	keys := make([]K, 0, m.size)
	for _, head := range m.buckets {
		for n := head; n != nil; n = n.next {
			keys = append(keys, n.key)
		}
	}
	return keys
}

// Clear removes every entry, keeping the current capacity.
func (m *HashMap[K, V]) Clear() {
	for i := range m.buckets {
		m.buckets[i] = nil
	}
	m.size = 0
}

func (m *HashMap[K, V]) index(key K) int {
	h := hash(key)
	return int(h) & (len(m.buckets) - 1)
}

func (m *HashMap[K, V]) scan(idx int, key K) (*bucketNode[K, V], bool) {
	for n := m.buckets[idx]; n != nil; n = n.next {
		if n.key == key {
			return n, true
		}
	}
	return nil, false
}

func (m *HashMap[K, V]) resize(newCap int) {
	old := m.buckets
	m.buckets = make([]*bucketNode[K, V], newCap)
	for _, head := range old {
		for n := head; n != nil; {
			next := n.next
			idx := m.index(n.key)
			n.next = m.buckets[idx]
			m.buckets[idx] = n
			n = next
		}
	}
}

func (m *HashMap[K, V]) maybeShrink() {
	if len(m.buckets) > DefaultCapacity && float64(m.size)/float64(len(m.buckets)) < 0.2 {
		m.resize(len(m.buckets) / 2)
	}
}

func hash[K comparable](key K) uint32 {
	return hashKey(key, 0)
}

func hashKey(key interface{}, seed uint32) uint32 {
	h := seed ^ 0x9e3779b9
	switch k := key.(type) {
	case string:
		for i := 0; i < len(k); i++ {
			h ^= uint32(k[i])
			h *= 0x01000193
		}
	case int:
		h ^= uint32(k)
		h *= 0x01000193
	case int32:
		h ^= uint32(k)
		h *= 0x01000193
	case int64:
		h ^= uint32(k)
		h *= 0x01000193
	case uint:
		h ^= uint32(k)
		h *= 0x01000193
	case uint32:
		h ^= k
		h *= 0x01000193
	case uint64:
		h ^= uint32(k)
		h *= 0x01000193
	case bool:
		if k {
			h ^= 1
		}
		h *= 0x01000193
	case float64:
		h ^= uint32(math.Float64bits(k))
		h *= 0x01000193
	default:
		s := fmt.Sprintf("%v", key)
		if len(s) > 0 {
			h ^= uint32(s[0])
		} else {
			h ^= 0xdeadbeef
		}
		h *= 0x01000193
	}
	return h
}
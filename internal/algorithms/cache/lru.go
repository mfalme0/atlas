// Package cache implements an LRU (Least Recently Used) cache.
//
// Operations:
//
//	Get: O(1)
//	Put: O(1)
//	Delete: O(1)
//
// Uses a doubly linked list for O(1) eviction and a hash table for O(1) lookup.
// When the cache exceeds capacity, the least recently used item is evicted.
//
// Use cases: web caches, DNS caches, database buffer pools, CDN caching,
// operating system page replacement.
package cache

// doubly linked list node for LRU
type lruNode[K comparable, V any] struct {
	key   K
	value V
	prev  *lruNode[K, V]
	next  *lruNode[K, V]
}

// LRUCache is a fixed-capacity LRU cache.
type LRUCache[K comparable, V any] struct {
	capacity int
	items    map[K]*lruNode[K, V]
	head     *lruNode[K, V]
	tail     *lruNode[K, V]
	size     int
}

// NewLRU creates a new LRU cache with the given capacity.
func NewLRU[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	head := &lruNode[K, V]{}
	tail := &lruNode[K, V]{}
	head.next = tail
	tail.prev = head

	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*lruNode[K, V]),
		head:     head,
		tail:     tail,
	}
}

// Get retrieves a value and marks it as recently used. O(1).
// Returns the value and true if found, or zero value and false if not.
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	if node, ok := c.items[key]; ok {
		c.moveToFront(node)
		return node.value, true
	}
	var zero V
	return zero, false
}

// Put inserts or updates a key-value pair. O(1).
// If the cache is full, the least recently used item is evicted.
func (c *LRUCache[K, V]) Put(key K, value V) {
	if node, ok := c.items[key]; ok {
		node.value = value
		c.moveToFront(node)
		return
	}

	if c.size >= c.capacity {
		evicted := c.removeTail()
		delete(c.items, evicted.key)
		c.size--
	}

	node := &lruNode[K, V]{key: key, value: value}
	c.addToFront(node)
	c.items[key] = node
	c.size++
}

// Delete removes a key-value pair. O(1).
// Returns true if the key was found and removed.
func (c *LRUCache[K, V]) Delete(key K) bool {
	if node, ok := c.items[key]; ok {
		c.removeNode(node)
		delete(c.items, key)
		c.size--
		return true
	}
	return false
}

// Contains returns true if the key exists without affecting access order. O(1).
func (c *LRUCache[K, V]) Contains(key K) bool {
	_, ok := c.items[key]
	return ok
}

// Size returns the number of items in the cache. O(1).
func (c *LRUCache[K, V]) Size() int {
	return c.size
}

// Capacity returns the maximum capacity. O(1).
func (c *LRUCache[K, V]) Capacity() int {
	return c.capacity
}

// IsEmpty returns true if the cache has no items. O(1).
func (c *LRUCache[K, V]) IsEmpty() bool {
	return c.size == 0
}

// Clear removes all items. O(n).
func (c *LRUCache[K, V]) Clear() {
	c.items = make(map[K]*lruNode[K, V])
	c.head.next = c.tail
	c.tail.prev = c.head
	c.size = 0
}

// Keys returns all keys in access order (most recent first). O(n).
func (c *LRUCache[K, V]) Keys() []K {
	keys := make([]K, 0, c.size)
	node := c.head.next
	for node != c.tail {
		keys = append(keys, node.key)
		node = node.next
	}
	return keys
}

// GetOldest returns the least recently used key-value pair without removing it. O(1).
func (c *LRUCache[K, V]) GetOldest() (K, V, bool) {
	if c.tail.prev == c.head {
		var zeroK K
		var zeroV V
		return zeroK, zeroV, false
	}
	oldest := c.tail.prev
	return oldest.key, oldest.value, true
}

func (c *LRUCache[K, V]) moveToFront(node *lruNode[K, V]) {
	c.removeNode(node)
	c.addToFront(node)
}

func (c *LRUCache[K, V]) addToFront(node *lruNode[K, V]) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

func (c *LRUCache[K, V]) removeNode(node *lruNode[K, V]) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (c *LRUCache[K, V]) removeTail() *lruNode[K, V] {
	node := c.tail.prev
	c.removeNode(node)
	return node
}

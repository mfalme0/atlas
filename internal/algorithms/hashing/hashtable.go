// Package hashing implements a hash table with separate chaining and dynamic resizing.
//
// Operations:
//
//	Put:    O(1) amortized
//	Get:    O(1) amortized
//	Delete: O(1) amortized
//	Resize: O(n) amortized
//
// The table maintains a load factor threshold of 0.75 and doubles capacity when exceeded.
// Collisions are handled via separate chaining (linked lists).
package hashing

const (
	defaultInitialCapacity = 16
	defaultLoadFactor      = 0.75
)

// Entry represents a key-value pair in the hash table.
type Entry[K comparable, V any] struct {
	Key   K
	Value V
	Next  *Entry[K, V]
}

// HashTable is a hash table with separate chaining.
type HashTable[K comparable, V any] struct {
	buckets    []*Entry[K, V]
	size       int
	capacity   int
	loadFactor float64
	hashFn     func(K) uint64
}

// New creates a new hash table with default settings.
func New[K comparable, V any](hashFn func(K) uint64) *HashTable[K, V] {
	return &HashTable[K, V]{
		buckets:    make([]*Entry[K, V], defaultInitialCapacity),
		capacity:   defaultInitialCapacity,
		loadFactor: defaultLoadFactor,
		hashFn:     hashFn,
	}
}

// NewWithCapacity creates a hash table with the specified initial capacity.
func NewWithCapacity[K comparable, V any](capacity int, hashFn func(K) uint64) *HashTable[K, V] {
	if capacity < 1 {
		capacity = 1
	}
	return &HashTable[K, V]{
		buckets:    make([]*Entry[K, V], capacity),
		capacity:   capacity,
		loadFactor: defaultLoadFactor,
		hashFn:     hashFn,
	}
}

// Size returns the number of key-value pairs. O(1).
func (ht *HashTable[K, V]) Size() int {
	return ht.size
}

// IsEmpty returns true if the table has no entries. O(1).
func (ht *HashTable[K, V]) IsEmpty() bool {
	return ht.size == 0
}

// Put inserts or updates a key-value pair. O(1) amortized.
func (ht *HashTable[K, V]) Put(key K, value V) {
	if float64(ht.size+1) > float64(ht.capacity)*ht.loadFactor {
		ht.resize(ht.capacity * 2)
	}

	bucket := ht.bucketIndex(key)
	entry := ht.buckets[bucket]

	for entry != nil {
		if entry.Key == key {
			entry.Value = value
			return
		}
		entry = entry.Next
	}

	newEntry := &Entry[K, V]{Key: key, Value: value, Next: ht.buckets[bucket]}
	ht.buckets[bucket] = newEntry
	ht.size++
}

// Get retrieves the value for a key. O(1) amortized.
func (ht *HashTable[K, V]) Get(key K) (V, bool) {
	bucket := ht.bucketIndex(key)
	entry := ht.buckets[bucket]

	for entry != nil {
		if entry.Key == key {
			return entry.Value, true
		}
		entry = entry.Next
	}

	var zero V
	return zero, false
}

// Delete removes a key-value pair. O(1) amortized.
func (ht *HashTable[K, V]) Delete(key K) bool {
	bucket := ht.bucketIndex(key)
	entry := ht.buckets[bucket]
	var prev *Entry[K, V]

	for entry != nil {
		if entry.Key == key {
			if prev != nil {
				prev.Next = entry.Next
			} else {
				ht.buckets[bucket] = entry.Next
			}
			ht.size--
			return true
		}
		prev = entry
		entry = entry.Next
	}
	return false
}

// Contains returns true if the key exists. O(1) amortized.
func (ht *HashTable[K, V]) Contains(key K) bool {
	_, ok := ht.Get(key)
	return ok
}

// Keys returns all keys. O(n).
func (ht *HashTable[K, V]) Keys() []K {
	keys := make([]K, 0, ht.size)
	for _, bucket := range ht.buckets {
		entry := bucket
		for entry != nil {
			keys = append(keys, entry.Key)
			entry = entry.Next
		}
	}
	return keys
}

// Values returns all values. O(n).
func (ht *HashTable[K, V]) Values() []V {
	values := make([]V, 0, ht.size)
	for _, bucket := range ht.buckets {
		entry := bucket
		for entry != nil {
			values = append(values, entry.Value)
			entry = entry.Next
		}
	}
	return values
}

// Clear removes all entries. O(n).
func (ht *HashTable[K, V]) Clear() {
	ht.buckets = make([]*Entry[K, V], ht.capacity)
	ht.size = 0
}

// LoadFactor returns the current load factor.
func (ht *HashTable[K, V]) LoadFactor() float64 {
	return float64(ht.size) / float64(ht.capacity)
}

func (ht *HashTable[K, V]) bucketIndex(key K) int {
	hash := ht.hashFn(key)
	return int(hash % uint64(ht.capacity))
}

func (ht *HashTable[K, V]) resize(newCapacity int) {
	oldBuckets := ht.buckets
	ht.buckets = make([]*Entry[K, V], newCapacity)
	ht.capacity = newCapacity
	ht.size = 0

	for _, bucket := range oldBuckets {
		entry := bucket
		for entry != nil {
			ht.Put(entry.Key, entry.Value)
			entry = entry.Next
		}
	}
}

// FNV-1a hash function for strings.
func HashString(s string) uint64 {
	hash := uint64(14695981039346656037)
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= 1099511628211
	}
	return hash
}

// Simple hash for integers.
func HashInt(key int) uint64 {
	h := uint64(key)
	h ^= h >> 16
	h *= 0x45d9f3b
	h ^= h >> 16
	return h
}

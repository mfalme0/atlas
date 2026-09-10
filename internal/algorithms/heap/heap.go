// Package heap implements binary heaps (min-heap and max-heap) and a priority queue.
//
// Binary Heap:
//
//	Insert:  O(log n)
//	Extract: O(log n)
//	Peek:    O(1)
//	Build from slice: O(n)
//
// Use cases: priority scheduling, median finding, graph algorithms (Dijkstra, Prim),
// event-driven simulation, top-K queries, job scheduling.
package heap

// Comparator returns negative if a < b, zero if a == b, positive if a > b.
type Comparator[T any] func(a, b T) int

// Heap is a binary heap that can be configured as min or max via the comparator.
type Heap[T any] struct {
	data []T
	cmp  Comparator[T]
}

// NewMinHeap creates a min-heap for a comparable type.
func NewMinHeap[T any](cmp Comparator[T]) *Heap[T] {
	return &Heap[T]{
		data: make([]T, 0),
		cmp:  cmp,
	}
}

// NewMaxHeap creates a max-heap by inverting the comparator.
func NewMaxHeap[T any](cmp Comparator[T]) *Heap[T] {
	inverted := func(a, b T) int { return cmp(b, a) }
	return &Heap[T]{
		data: make([]T, 0),
		cmp:  inverted,
	}
}

// NewMinHeapFromSlice builds a min-heap from an existing slice in O(n).
func NewMinHeapFromSlice[T any](items []T, cmp Comparator[T]) *Heap[T] {
	h := &Heap[T]{
		data: make([]T, len(items)),
		cmp:  cmp,
	}
	copy(h.data, items)
	h.heapify()
	return h
}

// NewMaxHeapFromSlice builds a max-heap from an existing slice in O(n).
func NewMaxHeapFromSlice[T any](items []T, cmp Comparator[T]) *Heap[T] {
	inverted := func(a, b T) int { return cmp(b, a) }
	h := &Heap[T]{
		data: make([]T, len(items)),
		cmp:  inverted,
	}
	copy(h.data, items)
	h.heapify()
	return h
}

// Size returns the number of elements. O(1).
func (h *Heap[T]) Size() int {
	return len(h.data)
}

// IsEmpty returns true if the heap has no elements. O(1).
func (h *Heap[T]) IsEmpty() bool {
	return len(h.data) == 0
}

// Peek returns the root element without removing it. O(1).
func (h *Heap[T]) Peek() (T, bool) {
	var zero T
	if len(h.data) == 0 {
		return zero, false
	}
	return h.data[0], true
}

// Insert adds an element and maintains the heap property. O(log n).
func (h *Heap[T]) Insert(value T) {
	h.data = append(h.data, value)
	h.siftUp(len(h.data) - 1)
}

// Extract removes and returns the root element. O(log n).
func (h *Heap[T]) Extract() (T, bool) {
	var zero T
	if len(h.data) == 0 {
		return zero, false
	}
	root := h.data[0]
	last := len(h.data) - 1
	h.data[0] = h.data[last]
	h.data = h.data[:last]
	if len(h.data) > 0 {
		h.siftDown(0)
	}
	return root, true
}

// ToSlice returns a copy of the internal data. O(n).
func (h *Heap[T]) ToSlice() []T {
	result := make([]T, len(h.data))
	copy(result, h.data)
	return result
}

func (h *Heap[T]) heapify() {
	for i := len(h.data)/2 - 1; i >= 0; i-- {
		h.siftDown(i)
	}
}

func (h *Heap[T]) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if h.cmp(h.data[i], h.data[parent]) >= 0 {
			break
		}
		h.data[i], h.data[parent] = h.data[parent], h.data[i]
		i = parent
	}
}

func (h *Heap[T]) siftDown(i int) {
	n := len(h.data)
	for {
		smallest := i
		left := 2*i + 1
		right := 2*i + 2

		if left < n && h.cmp(h.data[left], h.data[smallest]) < 0 {
			smallest = left
		}
		if right < n && h.cmp(h.data[right], h.data[smallest]) < 0 {
			smallest = right
		}
		if smallest == i {
			break
		}
		h.data[i], h.data[smallest] = h.data[smallest], h.data[i]
		i = smallest
	}
}

// PriorityQueue wraps a heap to support priority-based insertion.
type PriorityQueue[T any] struct {
	heap *Heap[pqItem[T]]
}

type pqItem[T any] struct {
	value    T
	priority int
}

// NewPriorityQueue creates a new priority queue (higher priority = dequeued first).
func NewPriorityQueue[T any]() *PriorityQueue[T] {
	cmp := func(a, b pqItem[T]) int {
		return b.priority - a.priority
	}
	return &PriorityQueue[T]{
		heap: NewMinHeap[pqItem[T]](cmp),
	}
}

// Enqueue adds an element with the given priority. O(log n).
func (pq *PriorityQueue[T]) Enqueue(value T, priority int) {
	pq.heap.Insert(pqItem[T]{value: value, priority: priority})
}

// Dequeue removes and returns the highest-priority element. O(log n).
func (pq *PriorityQueue[T]) Dequeue() (T, bool) {
	item, ok := pq.heap.Extract()
	if !ok {
		var zero T
		return zero, false
	}
	return item.value, true
}

// Peek returns the highest-priority element without removing it. O(1).
func (pq *PriorityQueue[T]) Peek() (T, bool) {
	item, ok := pq.heap.Peek()
	if !ok {
		var zero T
		return zero, false
	}
	return item.value, true
}

// Size returns the number of elements. O(1).
func (pq *PriorityQueue[T]) Size() int {
	return pq.heap.Size()
}

// IsEmpty returns true if the queue has no elements. O(1).
func (pq *PriorityQueue[T]) IsEmpty() bool {
	return pq.heap.IsEmpty()
}

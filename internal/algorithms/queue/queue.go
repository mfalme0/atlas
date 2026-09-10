// Package queue implements FIFO queue and circular queue.
//
// Queue: FIFO using a dynamic slice.
//
//	Enqueue: O(1) amortized
//	Dequeue: O(1) amortized
//
// CircularQueue: Fixed-size ring buffer.
//
//	Enqueue: O(1)
//	Dequeue: O(1)
//	Use cases: bounded buffers, I/O rings, rate limiters, producer-consumer patterns.
package queue

// Queue is a FIFO data structure.
type Queue[T any] struct {
	items []T
}

// New creates a new empty queue.
func New[T any]() *Queue[T] {
	return &Queue[T]{items: make([]T, 0)}
}

// Enqueue adds an element to the back. O(1) amortized.
func (q *Queue[T]) Enqueue(value T) {
	q.items = append(q.items, value)
}

// Dequeue removes and returns the front element. O(1) amortized.
func (q *Queue[T]) Dequeue() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	value := q.items[0]
	q.items = q.items[1:]
	return value, true
}

// Peek returns the front element without removing it. O(1).
func (q *Queue[T]) Peek() (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	return q.items[0], true
}

// Size returns the number of elements. O(1).
func (q *Queue[T]) Size() int {
	return len(q.items)
}

// IsEmpty returns true if the queue has no elements. O(1).
func (q *Queue[T]) IsEmpty() bool {
	return len(q.items) == 0
}

// Clear removes all elements. O(1).
func (q *Queue[T]) Clear() {
	q.items = q.items[:0]
}

// ToSlice returns a copy of the queue contents (front to back). O(n).
func (q *Queue[T]) ToSlice() []T {
	result := make([]T, len(q.items))
	copy(result, q.items)
	return result
}

// CircularQueue is a fixed-size ring buffer.
type CircularQueue[T any] struct {
	data  []T
	head  int
	tail  int
	size  int
	cap   int
}

// NewCircular creates a circular queue with the given capacity.
func NewCircular[T any](capacity int) *CircularQueue[T] {
	if capacity <= 0 {
		capacity = 1
	}
	return &CircularQueue[T]{
		data: make([]T, capacity),
		cap:  capacity,
	}
}

// Enqueue adds an element to the back. Returns false if full. O(1).
func (q *CircularQueue[T]) Enqueue(value T) bool {
	if q.size == q.cap {
		return false
	}
	q.data[q.tail] = value
	q.tail = (q.tail + 1) % q.cap
	q.size++
	return true
}

// Dequeue removes and returns the front element. Returns false if empty. O(1).
func (q *CircularQueue[T]) Dequeue() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}
	value := q.data[q.head]
	q.data[q.head] = zero
	q.head = (q.head + 1) % q.cap
	q.size--
	return value, true
}

// Peek returns the front element without removing it. O(1).
func (q *CircularQueue[T]) Peek() (T, bool) {
	var zero T
	if q.size == 0 {
		return zero, false
	}
	return q.data[q.head], true
}

// Size returns the number of elements. O(1).
func (q *CircularQueue[T]) Size() int {
	return q.size
}

// Capacity returns the maximum capacity. O(1).
func (q *CircularQueue[T]) Capacity() int {
	return q.cap
}

// IsEmpty returns true if the queue has no elements. O(1).
func (q *CircularQueue[T]) IsEmpty() bool {
	return q.size == 0
}

// IsFull returns true if the queue is at capacity. O(1).
func (q *CircularQueue[T]) IsFull() bool {
	return q.size == q.cap
}

// ToSlice returns the elements front to back. O(n).
func (q *CircularQueue[T]) ToSlice() []T {
	result := make([]T, 0, q.size)
	for i := 0; i < q.size; i++ {
		result = append(result, q.data[(q.head+i)%q.cap])
	}
	return result
}

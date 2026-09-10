// Package array implements a dynamic array (similar to Java's ArrayList or C++'s std::vector).
//
// The array automatically resizes when capacity is exceeded, doubling capacity on growth.
// All operations maintain contiguous memory layout for cache-friendly access.
//
// Complexity:
//
//	Access:   O(1)
//	Search:   O(n)
//	Insert:   O(1) amortized (O(n) worst case when resize needed)
//	Delete:   O(n)
//	Append:   O(1) amortized
//	Resize:   O(n)
package array

const initialCapacity = 16

// Array is a generic dynamic array.
type Array[T any] struct {
	data     []T
	size     int
	capacity int
}

// New creates a new empty Array with default capacity.
func New[T any]() *Array[T] {
	return &Array[T]{
		data:     make([]T, initialCapacity),
		size:     0,
		capacity: initialCapacity,
	}
}

// NewWithCapacity creates a new Array with the specified initial capacity.
func NewWithCapacity[T any](cap int) *Array[T] {
	if cap < 0 {
		cap = 0
	}
	return &Array[T]{
		data:     make([]T, cap),
		size:     0,
		capacity: cap,
	}
}

// Size returns the number of elements in the array.
func (a *Array[T]) Size() int {
	return a.size
}

// Capacity returns the current capacity of the array.
func (a *Array[T]) Capacity() int {
	return a.capacity
}

// IsEmpty returns true if the array contains no elements.
func (a *Array[T]) IsEmpty() bool {
	return a.size == 0
}

// Get returns the element at the given index.
// Returns an error if the index is out of bounds.
func (a *Array[T]) Get(index int) (T, bool) {
	var zero T
	if index < 0 || index >= a.size {
		return zero, false
	}
	return a.data[index], true
}

// Set replaces the element at the given index with the given value.
// Returns false if the index is out of bounds.
func (a *Array[T]) Set(index int, value T) bool {
	if index < 0 || index >= a.size {
		return false
	}
	a.data[index] = value
	return true
}

// Append adds an element to the end of the array. Amortized O(1).
func (a *Array[T]) Append(value T) {
	if a.size >= a.capacity {
		a.grow()
	}
	a.data[a.size] = value
	a.size++
}

// Insert inserts an element at the given index, shifting subsequent elements.
// Returns false if the index is out of bounds.
func (a *Array[T]) Insert(index int, value T) bool {
	if index < 0 || index > a.size {
		return false
	}
	if a.size >= a.capacity {
		a.grow()
	}
	copy(a.data[index+1:a.size+1], a.data[index:a.size])
	a.data[index] = value
	a.size++
	return true
}

// Remove removes and returns the element at the given index, shifting subsequent elements.
// Returns false if the index is out of bounds.
func (a *Array[T]) Remove(index int) (T, bool) {
	var zero T
	if index < 0 || index >= a.size {
		return zero, false
	}
	value := a.data[index]
	copy(a.data[index:a.size-1], a.data[index+1:a.size])
	a.size--
	a.data[a.size] = zero
	if a.size < a.capacity/4 && a.capacity > initialCapacity {
		a.shrink()
	}
	return value, true
}

// RemoveSwap removes the element at the given index by swapping with the last element.
// O(1) but does not preserve order.
func (a *Array[T]) RemoveSwap(index int) (T, bool) {
	var zero T
	if index < 0 || index >= a.size {
		return zero, false
	}
	value := a.data[index]
	a.size--
	a.data[index] = a.data[a.size]
	a.data[a.size] = zero
	if a.size < a.capacity/4 && a.capacity > initialCapacity {
		a.shrink()
	}
	return value, true
}

// Contains returns true if the array contains the given value.
// Requires T to support == comparison.
func (a *Array[T]) Contains(value T) bool {
	for i := 0; i < a.size; i++ {
		if any(a.data[i]) == any(value) {
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first occurrence of the given value, or -1.
func (a *Array[T]) IndexOf(value T) int {
	for i := 0; i < a.size; i++ {
		if any(a.data[i]) == any(value) {
			return i
		}
	}
	return -1
}

// Clear removes all elements from the array.
func (a *Array[T]) Clear() {
	var zero T
	for i := 0; i < a.size; i++ {
		a.data[i] = zero
	}
	a.size = 0
}

// ToSlice returns a copy of the underlying data as a Go slice.
func (a *Array[T]) ToSlice() []T {
	result := make([]T, a.size)
	copy(result, a.data[:a.size])
	return result
}

// ForEach calls the given function for each element in order.
func (a *Array[T]) ForEach(fn func(index int, value T)) {
	for i := 0; i < a.size; i++ {
		fn(i, a.data[i])
	}
}

func (a *Array[T]) grow() {
	newCap := a.capacity * 2
	if newCap == 0 {
		newCap = initialCapacity
	}
	newData := make([]T, newCap)
	copy(newData, a.data)
	a.data = newData
	a.capacity = newCap
}

func (a *Array[T]) shrink() {
	newCap := a.capacity / 2
	if newCap < initialCapacity {
		newCap = initialCapacity
	}
	newData := make([]T, newCap)
	copy(newData, a.data)
	a.data = newData
	a.capacity = newCap
}

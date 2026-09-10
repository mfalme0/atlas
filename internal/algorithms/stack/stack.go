// Package stack implements a LIFO (Last In, First Out) stack.
//
// All operations are O(1).
//
// Use cases: function call tracking, undo systems, expression evaluation,
// depth-first search, bracket matching, backtracking algorithms.
package stack

// Stack is a LIFO data structure backed by a slice.
type Stack[T any] struct {
	items []T
}

// New creates a new empty stack.
func New[T any]() *Stack[T] {
	return &Stack[T]{items: make([]T, 0)}
}

// Push adds an element to the top. O(1).
func (s *Stack[T]) Push(value T) {
	s.items = append(s.items, value)
}

// Pop removes and returns the top element. O(1).
// Returns false if the stack is empty.
func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

// Peek returns the top element without removing it. O(1).
func (s *Stack[T]) Peek() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

// Size returns the number of elements. O(1).
func (s *Stack[T]) Size() int {
	return len(s.items)
}

// IsEmpty returns true if the stack has no elements. O(1).
func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}

// Clear removes all elements. O(1).
func (s *Stack[T]) Clear() {
	s.items = s.items[:0]
}

// ToSlice returns a copy of the stack contents (bottom to top). O(n).
func (s *Stack[T]) ToSlice() []T {
	result := make([]T, len(s.items))
	copy(result, s.items)
	return result
}

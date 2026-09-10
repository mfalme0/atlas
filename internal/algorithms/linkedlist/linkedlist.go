// Package linkedlist implements singly and doubly linked lists.
//
// Singly Linked List:
//
//	Insert at head: O(1)
//	Insert at tail: O(1) with tail pointer
//	Delete by value: O(n)
//	Search: O(n)
//
// Doubly Linked List:
//
//	Insert at head: O(1)
//	Insert at tail: O(1)
//	Delete by node pointer: O(1)
//	Search: O(n)
package linkedlist

// SNode is a node in a singly linked list.
type SNode[T any] struct {
	Value T
	Next  *SNode[T]
}

// SinglyLinkedList is a singly linked list with head and tail pointers.
type SinglyLinkedList[T any] struct {
	Head   *SNode[T]
	Tail   *SNode[T]
	length int
}

// NewSingly creates a new empty singly linked list.
func NewSingly[T any]() *SinglyLinkedList[T] {
	return &SinglyLinkedList[T]{}
}

// Length returns the number of elements.
func (l *SinglyLinkedList[T]) Length() int {
	return l.length
}

// IsEmpty returns true if the list is empty.
func (l *SinglyLinkedList[T]) IsEmpty() bool {
	return l.length == 0
}

// PushFront inserts a value at the beginning. O(1).
func (l *SinglyLinkedList[T]) PushFront(value T) {
	node := &SNode[T]{Value: value, Next: l.Head}
	l.Head = node
	if l.Tail == nil {
		l.Tail = node
	}
	l.length++
}

// PushBack inserts a value at the end. O(1).
func (l *SinglyLinkedList[T]) PushBack(value T) {
	node := &SNode[T]{Value: value}
	if l.Tail == nil {
		l.Head = node
		l.Tail = node
	} else {
		l.Tail.Next = node
		l.Tail = node
	}
	l.length++
}

// PopFront removes and returns the first element. O(1).
func (l *SinglyLinkedList[T]) PopFront() (T, bool) {
	var zero T
	if l.Head == nil {
		return zero, false
	}
	value := l.Head.Value
	l.Head = l.Head.Next
	if l.Head == nil {
		l.Tail = nil
	}
	l.length--
	return value, true
}

// Find returns the first node containing the given value. O(n).
func (l *SinglyLinkedList[T]) Find(value T) *SNode[T] {
	for node := l.Head; node != nil; node = node.Next {
		if any(node.Value) == any(value) {
			return node
		}
	}
	return nil
}

// Remove removes the first occurrence of the given value. O(n).
func (l *SinglyLinkedList[T]) Remove(value T) bool {
	if l.Head == nil {
		return false
	}
	if any(l.Head.Value) == any(value) {
		l.PopFront()
		return true
	}
	prev := l.Head
	for curr := l.Head.Next; curr != nil; curr = curr.Next {
		if any(curr.Value) == any(value) {
			prev.Next = curr.Next
			if curr == l.Tail {
				l.Tail = prev
			}
			l.length--
			return true
		}
		prev = curr
	}
	return false
}

// ToSlice returns the list values as a slice.
func (l *SinglyLinkedList[T]) ToSlice() []T {
	result := make([]T, 0, l.length)
	for node := l.Head; node != nil; node = node.Next {
		result = append(result, node.Value)
	}
	return result
}

// Reverse reverses the list in place. O(n).
func (l *SinglyLinkedList[T]) Reverse() {
	var prev *SNode[T]
	curr := l.Head
	l.Tail = l.Head
	for curr != nil {
		next := curr.Next
		curr.Next = prev
		prev = curr
		curr = next
	}
	l.Head = prev
}

// DNode is a node in a doubly linked list.
type DNode[T any] struct {
	Value T
	Prev  *DNode[T]
	Next  *DNode[T]
}

// DoublyLinkedList is a doubly linked list.
type DoublyLinkedList[T any] struct {
	Head   *DNode[T]
	Tail   *DNode[T]
	length int
}

// NewDoubly creates a new empty doubly linked list.
func NewDoubly[T any]() *DoublyLinkedList[T] {
	return &DoublyLinkedList[T]{}
}

// Length returns the number of elements.
func (l *DoublyLinkedList[T]) Length() int {
	return l.length
}

// IsEmpty returns true if the list is empty.
func (l *DoublyLinkedList[T]) IsEmpty() bool {
	return l.length == 0
}

// PushFront inserts a value at the beginning. O(1).
func (l *DoublyLinkedList[T]) PushFront(value T) {
	node := &DNode[T]{Value: value, Next: l.Head}
	if l.Head != nil {
		l.Head.Prev = node
	}
	l.Head = node
	if l.Tail == nil {
		l.Tail = node
	}
	l.length++
}

// PushBack inserts a value at the end. O(1).
func (l *DoublyLinkedList[T]) PushBack(value T) {
	node := &DNode[T]{Value: value, Prev: l.Tail}
	if l.Tail != nil {
		l.Tail.Next = node
	}
	l.Tail = node
	if l.Head == nil {
		l.Head = node
	}
	l.length++
}

// PopFront removes and returns the first element. O(1).
func (l *DoublyLinkedList[T]) PopFront() (T, bool) {
	var zero T
	if l.Head == nil {
		return zero, false
	}
	value := l.Head.Value
	l.Head = l.Head.Next
	if l.Head != nil {
		l.Head.Prev = nil
	} else {
		l.Tail = nil
	}
	l.length--
	return value, true
}

// PopBack removes and returns the last element. O(1).
func (l *DoublyLinkedList[T]) PopBack() (T, bool) {
	var zero T
	if l.Tail == nil {
		return zero, false
	}
	value := l.Tail.Value
	l.Tail = l.Tail.Prev
	if l.Tail != nil {
		l.Tail.Next = nil
	} else {
		l.Head = nil
	}
	l.length--
	return value, true
}

// RemoveNode removes a specific node by pointer. O(1).
func (l *DoublyLinkedList[T]) RemoveNode(node *DNode[T]) bool {
	if node == nil {
		return false
	}
	if node.Prev != nil {
		node.Prev.Next = node.Next
	} else {
		l.Head = node.Next
	}
	if node.Next != nil {
		node.Next.Prev = node.Prev
	} else {
		l.Tail = node.Prev
	}
	l.length--
	return true
}

// Remove removes the first occurrence of the given value. O(n).
func (l *DoublyLinkedList[T]) Remove(value T) bool {
	for node := l.Head; node != nil; node = node.Next {
		if any(node.Value) == any(value) {
			return l.RemoveNode(node)
		}
	}
	return false
}

// ToSlice returns the list values as a slice.
func (l *DoublyLinkedList[T]) ToSlice() []T {
	result := make([]T, 0, l.length)
	for node := l.Head; node != nil; node = node.Next {
		result = append(result, node.Value)
	}
	return result
}

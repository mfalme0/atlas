package queue

import "testing"

func TestQueueEnqueueDequeue(t *testing.T) {
	q := New[int]()
	q.Enqueue(1)
	q.Enqueue(2)
	q.Enqueue(3)

	val, ok := q.Dequeue()
	if !ok || val != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", val, ok)
	}
	if q.Size() != 2 {
		t.Errorf("expected size 2, got %d", q.Size())
	}
}

func TestQueueFIFO(t *testing.T) {
	q := New[int]()
	input := []int{1, 2, 3, 4, 5}
	for _, v := range input {
		q.Enqueue(v)
	}

	for _, expected := range input {
		val, _ := q.Dequeue()
		if val != expected {
			t.Errorf("expected %d, got %d", expected, val)
		}
	}
}

func TestQueueDequeueEmpty(t *testing.T) {
	q := New[int]()
	_, ok := q.Dequeue()
	if ok {
		t.Error("expected false for empty queue")
	}
}

func TestQueuePeek(t *testing.T) {
	q := New[string]()
	q.Enqueue("a")
	q.Enqueue("b")

	val, ok := q.Peek()
	if !ok || val != "a" {
		t.Errorf("expected (a, true), got (%s, %v)", val, ok)
	}
	if q.Size() != 2 {
		t.Error("peek should not change size")
	}
}

func TestCircularQueueBasic(t *testing.T) {
	q := NewCircular[int](3)

	if !q.Enqueue(1) {
		t.Error("expected true")
	}
	if !q.Enqueue(2) {
		t.Error("expected true")
	}
	if !q.Enqueue(3) {
		t.Error("expected true")
	}
	if q.Enqueue(4) {
		t.Error("expected false when full")
	}
	if q.Size() != 3 {
		t.Errorf("expected size 3, got %d", q.Size())
	}

	val, ok := q.Dequeue()
	if !ok || val != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", val, ok)
	}

	if !q.Enqueue(4) {
		t.Error("should be able to enqueue after dequeue")
	}

	s := q.ToSlice()
	if len(s) != 3 || s[0] != 2 || s[1] != 3 || s[2] != 4 {
		t.Errorf("expected [2 3 4], got %v", s)
	}
}

func TestCircularQueueWrapAround(t *testing.T) {
	q := NewCircular[int](2)

	q.Enqueue(1)
	q.Enqueue(2)
	q.Dequeue()
	q.Enqueue(3)
	q.Dequeue()
	q.Enqueue(4)

	s := q.ToSlice()
	if len(s) != 2 || s[0] != 3 || s[1] != 4 {
		t.Errorf("expected [3 4], got %v", s)
	}
}

func TestCircularQueueIsEmpty(t *testing.T) {
	q := NewCircular[int](5)
	if !q.IsEmpty() {
		t.Error("expected empty")
	}
	q.Enqueue(1)
	if q.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestCircularQueueIsFull(t *testing.T) {
	q := NewCircular[int](2)
	if q.IsFull() {
		t.Error("expected not full")
	}
	q.Enqueue(1)
	q.Enqueue(2)
	if !q.IsFull() {
		t.Error("expected full")
	}
}

func TestCircularQueueCapacity(t *testing.T) {
	q := NewCircular[int](10)
	if q.Capacity() != 10 {
		t.Errorf("expected capacity 10, got %d", q.Capacity())
	}
}

func TestQueueClear(t *testing.T) {
	q := New[int]()
	q.Enqueue(1)
	q.Enqueue(2)
	q.Clear()
	if !q.IsEmpty() {
		t.Error("expected empty after clear")
	}
}

func BenchmarkQueueEnqueue(b *testing.B) {
	q := New[int]()
	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}
}

func BenchmarkQueueDequeue(b *testing.B) {
	q := New[int]()
	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Dequeue()
	}
}

func BenchmarkCircularEnqueue(b *testing.B) {
	q := NewCircular[int](b.N)
	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}
}

func BenchmarkCircularDequeue(b *testing.B) {
	q := NewCircular[int](b.N)
	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Dequeue()
	}
}

package heap

import "testing"

func intCmp(a, b int) int { return a - b }

func TestMinHeapInsertExtract(t *testing.T) {
	h := NewMinHeap[int](intCmp)
	h.Insert(5)
	h.Insert(3)
	h.Insert(7)
	h.Insert(1)

	val, ok := h.Extract()
	if !ok || val != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", val, ok)
	}
	val, _ = h.Extract()
	if val != 3 {
		t.Errorf("expected 3, got %d", val)
	}
}

func TestMaxHeapInsertExtract(t *testing.T) {
	h := NewMaxHeap[int](intCmp)
	h.Insert(5)
	h.Insert(3)
	h.Insert(7)
	h.Insert(1)

	val, ok := h.Extract()
	if !ok || val != 7 {
		t.Errorf("expected (7, true), got (%d, %v)", val, ok)
	}
	val, _ = h.Extract()
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}
}

func TestMinHeapFromSlice(t *testing.T) {
	h := NewMinHeapFromSlice([]int{5, 3, 7, 1, 4}, intCmp)

	expected := []int{1, 3, 4, 5, 7}
	for _, exp := range expected {
		val, _ := h.Extract()
		if val != exp {
			t.Errorf("expected %d, got %d", exp, val)
		}
	}
}

func TestMaxHeapFromSlice(t *testing.T) {
	h := NewMaxHeapFromSlice([]int{5, 3, 7, 1, 4}, intCmp)

	expected := []int{7, 5, 4, 3, 1}
	for _, exp := range expected {
		val, _ := h.Extract()
		if val != exp {
			t.Errorf("expected %d, got %d", exp, val)
		}
	}
}

func TestHeapPeek(t *testing.T) {
	h := NewMinHeap[int](intCmp)
	h.Insert(10)
	h.Insert(5)

	val, ok := h.Peek()
	if !ok || val != 5 {
		t.Errorf("expected (5, true), got (%d, %v)", val, ok)
	}
	if h.Size() != 2 {
		t.Error("peek should not change size")
	}
}

func TestHeapIsEmpty(t *testing.T) {
	h := NewMinHeap[int](intCmp)
	if !h.IsEmpty() {
		t.Error("expected empty")
	}
	h.Insert(1)
	if h.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestHeapExtractEmpty(t *testing.T) {
	h := NewMinHeap[int](intCmp)
	_, ok := h.Extract()
	if ok {
		t.Error("expected false for empty heap")
	}
}

func TestHeapSort(t *testing.T) {
	input := []int{10, 3, 8, 1, 7, 2, 9, 4, 6, 5}
	h := NewMinHeapFromSlice(input, intCmp)

	sorted := make([]int, 0, len(input))
	for !h.IsEmpty() {
		val, _ := h.Extract()
		sorted = append(sorted, val)
	}

	for i := 1; i < len(sorted); i++ {
		if sorted[i] < sorted[i-1] {
			t.Errorf("not sorted at index %d: %v", i, sorted)
			break
		}
	}
}

func TestPriorityQueue(t *testing.T) {
	pq := NewPriorityQueue[string]()
	pq.Enqueue("low priority", 1)
	pq.Enqueue("high priority", 10)
	pq.Enqueue("medium priority", 5)

	val, _ := pq.Dequeue()
	if val != "high priority" {
		t.Errorf("expected 'high priority', got '%s'", val)
	}
	val, _ = pq.Dequeue()
	if val != "medium priority" {
		t.Errorf("expected 'medium priority', got '%s'", val)
	}
	val, _ = pq.Dequeue()
	if val != "low priority" {
		t.Errorf("expected 'low priority', got '%s'", val)
	}
}

func TestPriorityQueueIsEmpty(t *testing.T) {
	pq := NewPriorityQueue[int]()
	if !pq.IsEmpty() {
		t.Error("expected empty")
	}
	pq.Enqueue(1, 1)
	if pq.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestPriorityQueuePeek(t *testing.T) {
	pq := NewPriorityQueue[int]()
	pq.Enqueue(100, 1)
	pq.Enqueue(200, 10)

	val, ok := pq.Peek()
	if !ok || val != 200 {
		t.Errorf("expected (200, true), got (%d, %v)", val, ok)
	}
}

func TestHeapLargeInput(t *testing.T) {
	h := NewMinHeap[int](intCmp)
	for i := 1000; i > 0; i-- {
		h.Insert(i)
	}

	prev := -1
	for !h.IsEmpty() {
		val, _ := h.Extract()
		if val <= prev {
			t.Fatalf("heap property violated: got %d after %d", val, prev)
		}
		prev = val
	}
}

func BenchmarkMinHeapInsert(b *testing.B) {
	h := NewMinHeap[int](intCmp)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Insert(i)
	}
}

func BenchmarkMinHeapExtract(b *testing.B) {
	h := NewMinHeap[int](intCmp)
	for i := 0; i < b.N; i++ {
		h.Insert(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Extract()
	}
}

func BenchmarkPriorityQueueEnqueue(b *testing.B) {
	pq := NewPriorityQueue[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.Enqueue(i, i)
	}
}

func BenchmarkPriorityQueueDequeue(b *testing.B) {
	pq := NewPriorityQueue[int]()
	for i := 0; i < b.N; i++ {
		pq.Enqueue(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.Dequeue()
	}
}

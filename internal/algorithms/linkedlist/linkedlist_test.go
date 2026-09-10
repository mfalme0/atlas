package linkedlist

import "testing"

func TestSinglyPushFront(t *testing.T) {
	l := NewSingly[int]()
	l.PushFront(3)
	l.PushFront(2)
	l.PushFront(1)

	s := l.ToSlice()
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", s)
	}
}

func TestSinglyPushBack(t *testing.T) {
	l := NewSingly[int]()
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)

	s := l.ToSlice()
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", s)
	}
}

func TestSinglyPopFront(t *testing.T) {
	l := NewSingly[int]()
	l.PushBack(1)
	l.PushBack(2)

	val, ok := l.PopFront()
	if !ok || val != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", val, ok)
	}
	if l.Length() != 1 {
		t.Errorf("expected length 1, got %d", l.Length())
	}
}

func TestSinglyPopFrontEmpty(t *testing.T) {
	l := NewSingly[int]()
	_, ok := l.PopFront()
	if ok {
		t.Error("expected false for empty list")
	}
}

func TestSinglyRemove(t *testing.T) {
	l := NewSingly[int]()
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)

	if !l.Remove(2) {
		t.Error("expected true for removing existing value")
	}
	s := l.ToSlice()
	if len(s) != 2 || s[0] != 1 || s[1] != 3 {
		t.Errorf("expected [1 3], got %v", s)
	}

	if l.Remove(99) {
		t.Error("expected false for removing non-existing value")
	}
}

func TestSinglyFind(t *testing.T) {
	l := NewSingly[int]()
	l.PushBack(10)
	l.PushBack(20)

	node := l.Find(20)
	if node == nil || node.Value != 20 {
		t.Error("expected to find 20")
	}

	if l.Find(99) != nil {
		t.Error("expected nil for non-existing value")
	}
}

func TestSinglyReverse(t *testing.T) {
	l := NewSingly[int]()
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)
	l.Reverse()

	s := l.ToSlice()
	if len(s) != 3 || s[0] != 3 || s[1] != 2 || s[2] != 1 {
		t.Errorf("expected [3 2 1], got %v", s)
	}
}

func TestSinglyLength(t *testing.T) {
	l := NewSingly[int]()
	if l.Length() != 0 {
		t.Errorf("expected 0, got %d", l.Length())
	}
	l.PushBack(1)
	if l.Length() != 1 {
		t.Errorf("expected 1, got %d", l.Length())
	}
}

func TestDoublyPushFront(t *testing.T) {
	l := NewDoubly[int]()
	l.PushFront(3)
	l.PushFront(2)
	l.PushFront(1)

	s := l.ToSlice()
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", s)
	}
}

func TestDoublyPushBack(t *testing.T) {
	l := NewDoubly[int]()
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)

	s := l.ToSlice()
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", s)
	}
}

func TestDoublyPopFront(t *testing.T) {
	l := NewDoubly[int]()
	l.PushBack(1)
	l.PushBack(2)

	val, ok := l.PopFront()
	if !ok || val != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", val, ok)
	}
}

func TestDoublyPopBack(t *testing.T) {
	l := NewDoubly[int]()
	l.PushBack(1)
	l.PushBack(2)

	val, ok := l.PopBack()
	if !ok || val != 2 {
		t.Errorf("expected (2, true), got (%d, %v)", val, ok)
	}
}

func TestDoublyRemoveNode(t *testing.T) {
	l := NewDoubly[int]()
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)

	mid := l.Head.Next
	l.RemoveNode(mid)

	s := l.ToSlice()
	if len(s) != 2 || s[0] != 1 || s[1] != 3 {
		t.Errorf("expected [1 3], got %v", s)
	}
}

func TestDoublyRemove(t *testing.T) {
	l := NewDoubly[int]()
	l.PushBack(10)
	l.PushBack(20)
	l.PushBack(30)

	if !l.Remove(20) {
		t.Error("expected true")
	}
	if l.Remove(99) {
		t.Error("expected false")
	}
	s := l.ToSlice()
	if len(s) != 2 || s[0] != 10 || s[1] != 30 {
		t.Errorf("expected [10 30], got %v", s)
	}
}

func TestDoublyIsEmpty(t *testing.T) {
	l := NewDoubly[int]()
	if !l.IsEmpty() {
		t.Error("expected empty")
	}
	l.PushBack(1)
	if l.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestSinglyRemoveHead(t *testing.T) {
	l := NewSingly[int]()
	l.PushBack(1)
	l.PushBack(2)
	l.Remove(1)

	if l.Head == nil || l.Head.Value != 2 {
		t.Error("expected head to be 2")
	}
}

func TestSinglyRemoveTail(t *testing.T) {
	l := NewSingly[int]()
	l.PushBack(1)
	l.PushBack(2)
	l.Remove(2)

	if l.Tail == nil || l.Tail.Value != 1 {
		t.Error("expected tail to be 1")
	}
}

func TestDoublyPushFrontThenBack(t *testing.T) {
	l := NewDoubly[int]()
	l.PushFront(2)
	l.PushFront(1)
	l.PushBack(3)

	s := l.ToSlice()
	if len(s) != 3 || s[0] != 1 || s[1] != 2 || s[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", s)
	}
}

func BenchmarkSinglyPushFront(b *testing.B) {
	l := NewSingly[int]()
	for i := 0; i < b.N; i++ {
		l.PushFront(i)
	}
}

func BenchmarkSinglyPushBack(b *testing.B) {
	l := NewSingly[int]()
	for i := 0; i < b.N; i++ {
		l.PushBack(i)
	}
}

func BenchmarkDoublyPushBack(b *testing.B) {
	l := NewDoubly[int]()
	for i := 0; i < b.N; i++ {
		l.PushBack(i)
	}
}

func BenchmarkDoublyPopBack(b *testing.B) {
	l := NewDoubly[int]()
	for i := 0; i < b.N; i++ {
		l.PushBack(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.PopBack()
	}
}

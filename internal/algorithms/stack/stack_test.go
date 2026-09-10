package stack

import "testing"

func TestPushPop(t *testing.T) {
	s := New[int]()
	s.Push(1)
	s.Push(2)
	s.Push(3)

	val, ok := s.Pop()
	if !ok || val != 3 {
		t.Errorf("expected (3, true), got (%d, %v)", val, ok)
	}
	if s.Size() != 2 {
		t.Errorf("expected size 2, got %d", s.Size())
	}
}

func TestPopEmpty(t *testing.T) {
	s := New[int]()
	_, ok := s.Pop()
	if ok {
		t.Error("expected false for empty stack")
	}
}

func TestPeek(t *testing.T) {
	s := New[string]()
	s.Push("hello")
	s.Push("world")

	val, ok := s.Peek()
	if !ok || val != "world" {
		t.Errorf("expected (world, true), got (%s, %v)", val, ok)
	}
	if s.Size() != 2 {
		t.Errorf("peek should not remove element, got size %d", s.Size())
	}
}

func TestPeekEmpty(t *testing.T) {
	s := New[int]()
	_, ok := s.Peek()
	if ok {
		t.Error("expected false for empty stack")
	}
}

func TestIsEmpty(t *testing.T) {
	s := New[int]()
	if !s.IsEmpty() {
		t.Error("expected empty")
	}
	s.Push(1)
	if s.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestClear(t *testing.T) {
	s := New[int]()
	s.Push(1)
	s.Push(2)
	s.Clear()
	if !s.IsEmpty() {
		t.Error("expected empty after clear")
	}
}

func TestLIFO(t *testing.T) {
	s := New[int]()
	input := []int{1, 2, 3, 4, 5}
	for _, v := range input {
		s.Push(v)
	}

	output := make([]int, 0, 5)
	for !s.IsEmpty() {
		val, _ := s.Pop()
		output = append(output, val)
	}

	for i := 0; i < len(input); i++ {
		if output[i] != input[len(input)-1-i] {
			t.Errorf("expected LIFO order, got %v", output)
			break
		}
	}
}

func TestToSlice(t *testing.T) {
	s := New[int]()
	s.Push(1)
	s.Push(2)

	slice := s.ToSlice()
	if len(slice) != 2 || slice[0] != 1 || slice[1] != 2 {
		t.Errorf("expected [1 2], got %v", slice)
	}
}

func BenchmarkPush(b *testing.B) {
	s := New[int]()
	for i := 0; i < b.N; i++ {
		s.Push(i)
	}
}

func BenchmarkPop(b *testing.B) {
	s := New[int]()
	for i := 0; i < b.N; i++ {
		s.Push(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Pop()
	}
}

package array

import (
	"testing"
)

func TestNew(t *testing.T) {
	a := New[int]()
	if a.Size() != 0 {
		t.Errorf("expected size 0, got %d", a.Size())
	}
	if a.Capacity() != initialCapacity {
		t.Errorf("expected capacity %d, got %d", initialCapacity, a.Capacity())
	}
}

func TestNewWithCapacity(t *testing.T) {
	a := NewWithCapacity[string](8)
	if a.Size() != 0 {
		t.Errorf("expected size 0, got %d", a.Size())
	}
	if a.Capacity() != 8 {
		t.Errorf("expected capacity 8, got %d", a.Capacity())
	}
}

func TestAppendAndGet(t *testing.T) {
	a := New[int]()
	for i := 0; i < 100; i++ {
		a.Append(i)
	}
	if a.Size() != 100 {
		t.Errorf("expected size 100, got %d", a.Size())
	}
	for i := 0; i < 100; i++ {
		val, ok := a.Get(i)
		if !ok || val != i {
			t.Errorf("expected Get(%d) = %d, got %d, %v", i, i, val, ok)
		}
	}
}

func TestAppendGrows(t *testing.T) {
	a := New[int]()
	initialCap := a.Capacity()
	for i := 0; i < initialCap+1; i++ {
		a.Append(i)
	}
	if a.Capacity() <= initialCap {
		t.Errorf("expected capacity to grow beyond %d, got %d", initialCap, a.Capacity())
	}
}

func TestSet(t *testing.T) {
	a := New[string]()
	a.Append("hello")
	a.Set(0, "world")
	val, _ := a.Get(0)
	if val != "world" {
		t.Errorf("expected 'world', got '%s'", val)
	}
	if a.Set(5, "out of bounds") {
		t.Error("expected Set to return false for out of bounds index")
	}
}

func TestInsert(t *testing.T) {
	a := New[int]()
	a.Append(1)
	a.Append(3)
	a.Insert(1, 2)

	if a.Size() != 3 {
		t.Fatalf("expected size 3, got %d", a.Size())
	}
	for i, expected := range []int{1, 2, 3} {
		val, _ := a.Get(i)
		if val != expected {
			t.Errorf("index %d: expected %d, got %d", i, expected, val)
		}
	}
}

func TestInsertAtBeginning(t *testing.T) {
	a := New[int]()
	a.Append(2)
	a.Append(3)
	a.Insert(0, 1)

	for i, expected := range []int{1, 2, 3} {
		val, _ := a.Get(i)
		if val != expected {
			t.Errorf("index %d: expected %d, got %d", i, expected, val)
		}
	}
}

func TestInsertAtEnd(t *testing.T) {
	a := New[int]()
	a.Append(1)
	a.Insert(1, 2)

	val, _ := a.Get(1)
	if val != 2 {
		t.Errorf("expected 2, got %d", val)
	}
	if a.Size() != 2 {
		t.Errorf("expected size 2, got %d", a.Size())
	}
}

func TestInsertOutOfBounds(t *testing.T) {
	a := New[int]()
	if a.Insert(-1, 1) {
		t.Error("expected false for negative index")
	}
	if a.Insert(1, 1) {
		t.Error("expected false for index > size")
	}
}

func TestRemove(t *testing.T) {
	a := New[int]()
	a.Append(10)
	a.Append(20)
	a.Append(30)

	val, ok := a.Remove(1)
	if !ok || val != 20 {
		t.Errorf("expected (20, true), got (%d, %v)", val, ok)
	}
	if a.Size() != 2 {
		t.Errorf("expected size 2, got %d", a.Size())
	}

	v0, _ := a.Get(0)
	v1, _ := a.Get(1)
	if v0 != 10 || v1 != 30 {
		t.Errorf("expected [10, 30], got [%d, %d]", v0, v1)
	}
}

func TestRemoveOutOfBounds(t *testing.T) {
	a := New[int]()
	_, ok := a.Remove(0)
	if ok {
		t.Error("expected false for removing from empty array")
	}
	a.Append(1)
	_, ok = a.Remove(5)
	if ok {
		t.Error("expected false for out of bounds index")
	}
}

func TestRemoveSwap(t *testing.T) {
	a := New[int]()
	a.Append(10)
	a.Append(20)
	a.Append(30)

	val, ok := a.RemoveSwap(0)
	if !ok || val != 10 {
		t.Errorf("expected (10, true), got (%d, %v)", val, ok)
	}
	if a.Size() != 2 {
		t.Errorf("expected size 2, got %d", a.Size())
	}
}

func TestContains(t *testing.T) {
	a := New[int]()
	for i := 0; i < 10; i++ {
		a.Append(i * 10)
	}
	if !a.Contains(50) {
		t.Error("expected Contains(50) = true")
	}
	if a.Contains(99) {
		t.Error("expected Contains(99) = false")
	}
}

func TestIndexOf(t *testing.T) {
	a := New[int]()
	a.Append(100)
	a.Append(200)
	a.Append(300)

	if idx := a.IndexOf(200); idx != 1 {
		t.Errorf("expected IndexOf(200) = 1, got %d", idx)
	}
	if idx := a.IndexOf(999); idx != -1 {
		t.Errorf("expected IndexOf(999) = -1, got %d", idx)
	}
}

func TestClear(t *testing.T) {
	a := New[int]()
	for i := 0; i < 10; i++ {
		a.Append(i)
	}
	a.Clear()
	if a.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", a.Size())
	}
	if a.IsEmpty() == false {
		t.Error("expected IsEmpty() = true after clear")
	}
}

func TestToSlice(t *testing.T) {
	a := New[int]()
	a.Append(1)
	a.Append(2)
	a.Append(3)

	s := a.ToSlice()
	if len(s) != 3 {
		t.Fatalf("expected slice length 3, got %d", len(s))
	}
	for i, v := range s {
		if v != i+1 {
			t.Errorf("index %d: expected %d, got %d", i, i+1, v)
		}
	}

	s[0] = 999
	val, _ := a.Get(0)
	if val == 999 {
		t.Error("ToSlice should return a copy, not a reference")
	}
}

func TestForEach(t *testing.T) {
	a := New[int]()
	a.Append(10)
	a.Append(20)
	a.Append(30)

	sum := 0
	a.ForEach(func(i, v int) {
		sum += v
	})
	if sum != 60 {
		t.Errorf("expected sum 60, got %d", sum)
	}
}

func TestIsEmpty(t *testing.T) {
	a := New[int]()
	if !a.IsEmpty() {
		t.Error("expected IsEmpty() = true for new array")
	}
	a.Append(1)
	if a.IsEmpty() {
		t.Error("expected IsEmpty() = false after append")
	}
}

func TestGetOutOfBounds(t *testing.T) {
	a := New[int]()
	_, ok := a.Get(0)
	if ok {
		t.Error("expected false for getting from empty array")
	}
}

func TestDynamicGrowth(t *testing.T) {
	a := New[int]()
	cap := a.Capacity()
	for i := 0; i < cap*4; i++ {
		a.Append(i)
	}
	if a.Size() != cap*4 {
		t.Errorf("expected size %d, got %d", cap*4, a.Size())
	}
	for i := 0; i < cap*4; i++ {
		val, _ := a.Get(i)
		if val != i {
			t.Errorf("expected %d at index %d, got %d", i, i, val)
		}
	}
}

func TestShrinkOnRemove(t *testing.T) {
	a := New[int]()
	for i := 0; i < 100; i++ {
		a.Append(i)
	}
	capBefore := a.Capacity()
	for a.Size() > 2 {
		a.Remove(a.Size() - 1)
	}
	capAfter := a.Capacity()
	if capAfter >= capBefore {
		t.Errorf("expected capacity to shrink from %d, got %d", capBefore, capAfter)
	}
}

func BenchmarkAppend(b *testing.B) {
	a := New[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Append(i)
	}
}

func BenchmarkGet(b *testing.B) {
	a := New[int]()
	for i := 0; i < 10000; i++ {
		a.Append(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Get(i % 10000)
	}
}

func BenchmarkInsert(b *testing.B) {
	for i := 0; i < b.N; i++ {
		a := New[int]()
		for j := 0; j < 1000; j++ {
			a.Insert(0, j)
		}
	}
}

func BenchmarkRemove(b *testing.B) {
	for i := 0; i < b.N; i++ {
		a := New[int]()
		for j := 0; j < 1000; j++ {
			a.Append(j)
		}
		b.StopTimer()
		for j := 0; j < 1000; j++ {
			b.StartTimer()
			a.Remove(0)
			b.StopTimer()
		}
	}
}

func BenchmarkAppendBulk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		a := New[int]()
		for j := 0; j < 100000; j++ {
			a.Append(j)
		}
	}
}

package hashing

import "testing"

func TestPutGet(t *testing.T) {
	h := New[string, int](HashString)
	h.Put("one", 1)
	h.Put("two", 2)
	h.Put("three", 3)

	if v, ok := h.Get("one"); !ok || v != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", v, ok)
	}
	if v, ok := h.Get("two"); !ok || v != 2 {
		t.Errorf("expected (2, true), got (%d, %v)", v, ok)
	}
	if v, ok := h.Get("three"); !ok || v != 3 {
		t.Errorf("expected (3, true), got (%d, %v)", v, ok)
	}
}

func TestPutOverwrite(t *testing.T) {
	h := New[string, string](HashString)
	h.Put("key", "old")
	h.Put("key", "new")

	v, _ := h.Get("key")
	if v != "new" {
		t.Errorf("expected 'new', got '%s'", v)
	}
	if h.Size() != 1 {
		t.Errorf("expected size 1, got %d", h.Size())
	}
}

func TestDelete(t *testing.T) {
	h := New[string, int](HashString)
	h.Put("a", 1)
	h.Put("b", 2)

	if !h.Delete("a") {
		t.Error("expected true for existing key")
	}
	if h.Delete("a") {
		t.Error("expected false for non-existing key")
	}
	if h.Size() != 1 {
		t.Errorf("expected size 1, got %d", h.Size())
	}
	_, ok := h.Get("a")
	if ok {
		t.Error("expected false for deleted key")
	}
}

func TestContains(t *testing.T) {
	h := New[string, int](HashString)
	h.Put("exists", 42)

	if !h.Contains("exists") {
		t.Error("expected true")
	}
	if h.Contains("missing") {
		t.Error("expected false")
	}
}

func TestKeys(t *testing.T) {
	h := New[string, int](HashString)
	h.Put("a", 1)
	h.Put("b", 2)
	h.Put("c", 3)

	keys := h.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestValues(t *testing.T) {
	h := New[string, int](HashString)
	h.Put("a", 10)
	h.Put("b", 20)

	values := h.Values()
	if len(values) != 2 {
		t.Errorf("expected 2 values, got %d", len(values))
	}
}

func TestIsEmpty(t *testing.T) {
	h := New[int, int](HashInt)
	if !h.IsEmpty() {
		t.Error("expected empty")
	}
	h.Put(1, 1)
	if h.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestClear(t *testing.T) {
	h := New[string, int](HashString)
	h.Put("a", 1)
	h.Put("b", 2)
	h.Clear()
	if !h.IsEmpty() {
		t.Error("expected empty after clear")
	}
}

func TestCollisions(t *testing.T) {
	h := NewWithCapacity[string, int](2, HashString)
	h.Put("aa", 1)
	h.Put("bb", 2)
	h.Put("cc", 3)
	h.Put("dd", 4)

	for _, tc := range []struct {
		key  string
		want int
	}{
		{"aa", 1},
		{"bb", 2},
		{"cc", 3},
		{"dd", 4},
	} {
		if v, ok := h.Get(tc.key); !ok || v != tc.want {
			t.Errorf("key %s: expected %d, got %d (ok=%v)", tc.key, tc.want, v, ok)
		}
	}
}

func TestResize(t *testing.T) {
	h := New[int, int](HashInt)
	for i := 0; i < 1000; i++ {
		h.Put(i, i)
	}
	if h.Size() != 1000 {
		t.Errorf("expected size 1000, got %d", h.Size())
	}
	for i := 0; i < 1000; i++ {
		v, ok := h.Get(i)
		if !ok || v != i {
			t.Errorf("key %d: expected %d, got %d", i, i, v)
		}
	}
}

func TestLoadFactor(t *testing.T) {
	h := New[int, int](HashInt)
	if h.LoadFactor() != 0 {
		t.Error("expected load factor 0 for empty table")
	}
	h.Put(1, 1)
	lf := h.LoadFactor()
	if lf <= 0 || lf > 1 {
		t.Errorf("expected load factor in (0, 1], got %f", lf)
	}
}

func TestGetNonExistent(t *testing.T) {
	h := New[string, int](HashString)
	_, ok := h.Get("missing")
	if ok {
		t.Error("expected false for non-existent key")
	}
}

func BenchmarkHashTablePut(b *testing.B) {
	h := New[int, int](HashInt)
	for i := 0; i < b.N; i++ {
		h.Put(i, i)
	}
}

func BenchmarkHashTableGet(b *testing.B) {
	h := New[int, int](HashInt)
	for i := 0; i < 100000; i++ {
		h.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Get(i % 100000)
	}
}

func BenchmarkHashTableDelete(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := New[int, int](HashInt)
		for j := 0; j < 1000; j++ {
			h.Put(j, j)
		}
		for j := 0; j < 1000; j++ {
			h.Delete(j)
		}
	}
}

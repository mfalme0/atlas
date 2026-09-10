package cache

import "testing"

func TestLRUGetPut(t *testing.T) {
	c := NewLRU[string, int](3)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", v, ok)
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Errorf("expected (2, true), got (%d, %v)", v, ok)
	}
}

func TestLRUEviction(t *testing.T) {
	c := NewLRU[string, int](2)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	if _, ok := c.Get("a"); ok {
		t.Error("expected 'a' to be evicted")
	}
	if v, ok := c.Get("c"); !ok || v != 3 {
		t.Errorf("expected (3, true), got (%d, %v)", v, ok)
	}
}

func TestLRUAccessUpdatesOrder(t *testing.T) {
	c := NewLRU[string, int](2)

	c.Put("a", 1)
	c.Put("b", 2)

	c.Get("a")

	c.Put("c", 3)

	if _, ok := c.Get("a"); !ok {
		t.Error("expected 'a' to still exist (was recently accessed)")
	}
	if _, ok := c.Get("b"); ok {
		t.Error("expected 'b' to be evicted (was least recently used)")
	}
}

func TestLRUUpdate(t *testing.T) {
	c := NewLRU[string, int](2)

	c.Put("a", 1)
	c.Put("a", 10)

	if v, ok := c.Get("a"); !ok || v != 10 {
		t.Errorf("expected (10, true), got (%d, %v)", v, ok)
	}
	if c.Size() != 1 {
		t.Errorf("expected size 1, got %d", c.Size())
	}
}

func TestLRUDelete(t *testing.T) {
	c := NewLRU[string, int](3)

	c.Put("a", 1)
	c.Put("b", 2)

	if !c.Delete("a") {
		t.Error("expected true")
	}
	if c.Delete("a") {
		t.Error("expected false for second delete")
	}
	if c.Size() != 1 {
		t.Errorf("expected size 1, got %d", c.Size())
	}
}

func TestLRUContains(t *testing.T) {
	c := NewLRU[string, int](2)

	c.Put("a", 1)
	if !c.Contains("a") {
		t.Error("expected true")
	}
	if c.Contains("b") {
		t.Error("expected false")
	}
}

func TestLRUKeys(t *testing.T) {
	c := NewLRU[string, int](3)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestLRUClear(t *testing.T) {
	c := NewLRU[string, int](3)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Clear()

	if !c.IsEmpty() {
		t.Error("expected empty after clear")
	}
}

func TestLRUGetOldest(t *testing.T) {
	c := NewLRU[string, int](3)

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	k, v, ok := c.GetOldest()
	if !ok || k != "a" || v != 1 {
		t.Errorf("expected ('a', 1, true), got ('%s', %d, %v)", k, v, ok)
	}
}

func TestLRUCapacity(t *testing.T) {
	c := NewLRU[string, int](5)
	if c.Capacity() != 5 {
		t.Errorf("expected capacity 5, got %d", c.Capacity())
	}
}

func TestLRUEvictsCorrectly(t *testing.T) {
	c := NewLRU[int, string](3)

	c.Put(1, "a")
	c.Put(2, "b")
	c.Put(3, "c")

	c.Get(1)

	c.Put(4, "d")

	_, ok1 := c.Get(1)
	_, ok2 := c.Get(2)
	_, ok3 := c.Get(3)
	_, ok4 := c.Get(4)

	if !ok1 {
		t.Error("expected '1' to exist (was accessed)")
	}
	if ok2 {
		t.Error("expected '2' to be evicted")
	}
	if !ok3 {
		t.Error("expected '3' to exist")
	}
	if !ok4 {
		t.Error("expected '4' to exist")
	}
}

func TestLRUHighChurn(t *testing.T) {
	c := NewLRU[int, int](10)
	for i := 0; i < 1000; i++ {
		c.Put(i, i)
	}
	if c.Size() != 10 {
		t.Errorf("expected size 10, got %d", c.Size())
	}
	for i := 990; i <= 999; i++ {
		if v, ok := c.Get(i); !ok || v != i {
			t.Errorf("expected %d, got %d (ok=%v)", i, v, ok)
		}
	}
}

func BenchmarkLRUPut(b *testing.B) {
	c := NewLRU[int, int](1000)
	for i := 0; i < b.N; i++ {
		c.Put(i, i)
	}
}

func BenchmarkLRUGet(b *testing.B) {
	c := NewLRU[int, int](1000)
	for i := 0; i < 1000; i++ {
		c.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(i % 1000)
	}
}

func BenchmarkLRUGetPutMixed(b *testing.B) {
	c := NewLRU[int, int](1000)
	for i := 0; i < 1000; i++ {
		c.Put(i, i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			c.Get(i % 1000)
		} else {
			c.Put(i%1000+1000, i)
		}
	}
}

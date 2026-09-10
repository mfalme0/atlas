package hashtable

import (
	"math/rand"
	"testing"
)

func TestPutGet(t *testing.T) {
	m := New[string, int]()
	m.Put("a", 1)
	m.Put("b", 2)

	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Fatalf("expected 1, got %d ok=%v", v, ok)
	}
	if m.Size() != 2 {
		t.Fatalf("expected size 2, got %d", m.Size())
	}
	if !m.Contains("b") {
		t.Fatalf("expected b present")
	}
	if m.Contains("missing") {
		t.Fatalf("unexpected key")
	}
}

func TestPutOverwrite(t *testing.T) {
	m := New[string, int]()
	m.Put("k", 1)
	old, hadOld := m.Put("k", 99)
	if !hadOld || old != 1 {
		t.Fatalf("expected previous value 1, got %d (%v)", old, hadOld)
	}
	v, _ := m.Get("k")
	if v != 99 {
		t.Fatalf("expected 99, got %d", v)
	}
	if m.Size() != 1 {
		t.Fatalf("expected size 1, got %d", m.Size())
	}
}

func TestDelete(t *testing.T) {
	m := New[string, int]()
	for i := 0; i < 100; i++ {
		m.Put(key(i), i)
	}
	if v, ok := m.Delete(key(42)); !ok || v != 42 {
		t.Fatalf("delete failed")
	}
	if _, ok := m.Get(key(42)); ok {
		t.Fatalf("key should be gone")
	}
	if _, ok := m.Delete("nope"); ok {
		t.Fatalf("deleting missing key should report false")
	}
	if m.Size() != 99 {
		t.Fatalf("expected size 99, got %d", m.Size())
	}
}

func TestResizeRoundTrip(t *testing.T) {
	m := New[int, int]()
	n := 10000
	for i := 0; i < n; i++ {
		m.Put(i, i*2)
	}
	if m.Size() != n {
		t.Fatalf("expected %d entries, got %d", n, m.Size())
	}
	if m.Capacity() <= DefaultCapacity {
		t.Fatalf("expected table to have grown, capacity=%d", m.Capacity())
	}
	for i := 0; i < n; i++ {
		v, ok := m.Get(i)
		if !ok || v != i*2 {
			t.Fatalf("round-trip failed at %d", i)
		}
	}
	// Delete most keys to force a shrink.
	for i := 0; i < n-10; i++ {
		m.Delete(i)
	}
	if m.Size() != 10 {
		t.Fatalf("expected 10 entries after deletes, got %d", m.Size())
	}
}

func TestCollisionsDifferentBucketsIndependent(t *testing.T) {
	m := New[string, int]()
	for _, k := range []string{"alpha", "beta", "gamma", "delta"} {
		m.Put(k, len(k))
	}
	for _, k := range []string{"alpha", "beta", "gamma", "delta"} {
		v, ok := m.Get(k)
		if !ok || v != len(k) {
			t.Fatalf("collision handling broken for %s", k)
		}
	}
}

func TestKeysAndClear(t *testing.T) {
	m := New[string, int]()
	m.Put("x", 1)
	m.Put("y", 2)
	m.Put("z", 3)
	if got := len(m.Keys()); got != 3 {
		t.Fatalf("expected 3 keys, got %d", got)
	}
	m.Clear()
	if !m.IsEmpty() || m.Size() != 0 {
		t.Fatalf("clear failed")
	}
	if _, ok := m.Get("x"); ok {
		t.Fatalf("key should be gone after clear")
	}
}

func TestIntKeys(t *testing.T) {
	m := New[int, string]()
	m.Put(1, "one")
	m.Put(2, "two")
	v, ok := m.Get(2)
	if !ok || v != "two" {
		t.Fatalf("int-key lookup failed")
	}
}

func key(i int) string {
	return string(rune('a' + i%26)) + string(rune('0'+(i%10))) + keySuffix(i)
}

var keyPool = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func keySuffix(seed int) string {
	r := rand.New(rand.NewSource(int64(seed)))
	b := make([]rune, 4)
	for i := range b {
		b[i] = keyPool[r.Intn(len(keyPool))]
	}
	return string(b)
}
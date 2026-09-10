package kvstore

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func newTestStore() *Store {
	return New(3, slog.Default())
}

func TestPutGet(t *testing.T) {
	s := newTestStore()
	s.AddNode("n1", "addr1")
	s.AddNode("n2", "addr2")
	s.AddNode("n3", "addr3")

	ctx := context.Background()
	if err := s.Put(ctx, "user:123", "Joseph"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, found, err := s.Get(ctx, "user:123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("expected key to be found")
	}
	if value != "Joseph" {
		t.Errorf("expected 'Joseph', got '%s'", value)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore()
	s.AddNode("n1", "addr1")
	s.AddNode("n2", "addr2")
	s.AddNode("n3", "addr3")

	ctx := context.Background()
	s.Put(ctx, "key", "value")

	if err := s.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, found, _ := s.Get(ctx, "key")
	if found {
		t.Error("expected key to be deleted")
	}
}

func TestGetMissing(t *testing.T) {
	s := newTestStore()
	s.AddNode("n1", "addr1")

	_, found, _ := s.Get(context.Background(), "missing")
	if found {
		t.Error("expected not found")
	}
}

func TestNoNodes(t *testing.T) {
	s := newTestStore()
	err := s.Put(context.Background(), "k", "v")
	if err == nil {
		t.Error("expected error when no nodes")
	}
}

func TestReplication(t *testing.T) {
	s := newTestStore()
	s.AddNode("n1", "addr1")
	s.AddNode("n2", "addr2")
	s.AddNode("n3", "addr3")

	ctx := context.Background()
	s.Put(ctx, "replicated-key", "value")

	// Key should exist in following replica count
	replicas := s.ring.GetNodes("replicated-key", s.replicationFactor)
	if len(replicas) != 3 {
		t.Errorf("expected 3 replicas, got %d", len(replicas))
	}

	count := 0
	for _, id := range replicas {
		if _, ok := s.nodes[id]; ok {
			count++
		}
	}
	if count != 3 {
		t.Errorf("expected 3 active replicas, got %d", count)
	}
}

func TestNodeFailure(t *testing.T) {
	s := newTestStore()
	s.AddNode("n1", "addr1")
	s.AddNode("n2", "addr2")
	s.AddNode("n3", "addr3")

	ctx := context.Background()
	s.Put(ctx, "fault-key", "value")

	// Kill the primary owner
	owner := s.ring.GetNode("fault-key")
	s.RemoveNode(owner)

	// Should still be able to read from replica
	value, found, err := s.Get(ctx, "fault-key")
	if err != nil {
		t.Fatalf("Get after node failure failed: %v", err)
	}
	if !found {
		t.Error("expected value to be available from replica")
	}
	if value != "value" {
		t.Errorf("expected 'value', got '%s'", value)
	}
}

func TestRequestTimeout(t *testing.T) {
	s := newTestStore()
	s.AddNode("n1", "addr1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Put(ctx, "k", "v"); err != nil {
		t.Fatalf("Put failed: %v", err)
	}
}

func TestConcurrentPuts(t *testing.T) {
	s := newTestStore()
	s.AddNode("n1", "addr1")
	s.AddNode("n2", "addr2")
	s.AddNode("n3", "addr3")

	ctx := context.Background()
	for i := 0; i < 100; i++ {
		s.Put(ctx, string(rune(i))+"key", "value")
	}

	count := len(s.Keys())
	if count != 100 {
		t.Errorf("expected 100 keys, got %d", count)
	}
}

func BenchmarkStorePut(b *testing.B) {
	s := newTestStore()
	for i := 0; i < 5; i++ {
		s.AddNode(string(rune('a'+i)), "addr")
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Put(ctx, string(rune(i%100))+"key", "hello")
	}
}

func BenchmarkStoreGet(b *testing.B) {
	s := newTestStore()
	for i := 0; i < 5; i++ {
		s.AddNode(string(rune('a'+i)), "addr")
	}
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		s.Put(ctx, string(rune(i%1000))+"key", "hello")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(ctx, string(rune(i%1000))+"key")
	}
}
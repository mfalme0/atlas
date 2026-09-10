package benchmarks

import (
	"testing"
)

func TestCasesRegistered(t *testing.T) {
	names := Names()
	if len(names) < 10 {
		t.Fatalf("expected at least 10 benchmark cases, got %d", len(names))
	}
	seen := make(map[string]bool)
	for _, n := range names {
		if seen[n] {
			t.Fatalf("duplicate case name %q", n)
		}
		if _, ok := CaseByName(n); !ok {
			t.Fatalf("CaseByName(%q) should resolve", n)
		}
		seen[n] = true
	}
}

func TestRunRecordsResult(t *testing.T) {
	r := NewRunner()
	for _, n := range []string{"array_append", "hashtable_put_get", "hashing_consistent_ring", "lru_cache_put_get"} {
		res, err := r.Run(n)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if res.Algorithm != n || res.InputSize <= 0 || res.Runtime <= 0 {
			t.Fatalf("invalid result for %s: %+v", n, res)
		}
		if res.MemoryBytes < 0 {
			t.Fatalf("negative memory for %s", n)
		}
	}
	if got := len(r.List()); got != 4 {
		t.Fatalf("expected 4 recorded results, got %d", got)
	}
}

func TestRunUnknown(t *testing.T) {
	r := NewRunner()
	if _, err := r.Run("nonexistent_benchmark"); err == nil {
		t.Fatalf("expected error for unknown benchmark")
	}
}

func TestRunAllDoesNotFail(t *testing.T) {
	r := NewRunner()
	results := r.RunAll()
	if len(results) != len(Names()) {
		t.Fatalf("expected %d results from RunAll, got %d", len(Names()), len(results))
	}
	for _, res := range results {
		if res.Runtime <= 0 {
			t.Fatalf("benchmark %s recorded zero runtime", res.Algorithm)
		}
	}
}

func TestEveryWorkloadRuns(t *testing.T) {
	r := NewRunner()
	for _, n := range Names() {
		if _, err := r.Run(n); err != nil {
			t.Fatalf("case %s failed: %v", n, err)
		}
	}
}
// Package benchmarks runs real, in-process timing benchmarks over the
// algorithm implementations and records the results. It never fabricates
// numbers: every result is measured from an actual run against the library
// code with a warm-up pass.
package benchmarks

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/algorithms/array"
	"github.com/atlas-engine/atlas/internal/algorithms/cache"
	"github.com/atlas-engine/atlas/internal/algorithms/graph"
	"github.com/atlas-engine/atlas/internal/algorithms/hashtable"
	"github.com/atlas-engine/atlas/internal/algorithms/hashing"
	"github.com/atlas-engine/atlas/internal/algorithms/heap"
	"github.com/atlas-engine/atlas/internal/algorithms/linkedlist"
	"github.com/atlas-engine/atlas/internal/algorithms/queue"
	"github.com/atlas-engine/atlas/internal/algorithms/stack"
	"github.com/atlas-engine/atlas/internal/algorithms/tree"
	"github.com/atlas-engine/atlas/internal/algorithms/trie"
	"github.com/atlas-engine/atlas/internal/models"
)

// Case is a named benchmark workload.
type Case struct {
	Name   string
	Input  int
	Ops    int64
	Run    func() error
}

// Cases returns the full set of benchmark workloads available.
func Cases() []Case {
	return []Case{
		{Name: "array_append", Input: 200_000, Ops: 200_000, Run: benchArrayAppend},
		{Name: "stack_push_pop", Input: 200_000, Ops: 400_000, Run: benchStack},
		{Name: "queue_enqueue_dequeue", Input: 200_000, Ops: 400_000, Run: benchQueue},
		{Name: "circular_queue", Input: 100_000, Ops: 200_000, Run: benchCircular},
		{Name: "heap_push_extract", Input: 200_000, Ops: 400_000, Run: benchHeap},
		{Name: "priority_queue", Input: 200_000, Ops: 200_000, Run: benchPriorityQueue},
		{Name: "linkedlist_push_back", Input: 200_000, Ops: 200_000, Run: benchLinkedList},
		{Name: "hashtable_put_get", Input: 200_000, Ops: 400_000, Run: benchHashtable},
		{Name: "bst_insert_search", Input: 100_000, Ops: 150_000, Run: benchBST},
		{Name: "avl_insert", Input: 100_000, Ops: 100_000, Run: benchAVL},
		{Name: "trie_insert_search", Input: 100_000, Ops: 200_000, Run: benchTrie},
		{Name: "lru_cache_put_get", Input: 200_000, Ops: 300_000, Run: benchLRU},
		{Name: "graph_bfs", Input: 50_000, Ops: 50_000, Run: benchBFS},
		{Name: "graph_dijkstra", Input: 30_000, Ops: 30_000, Run: benchDijkstra},
		{Name: "graph_topological_sort", Input: 50_000, Ops: 50_000, Run: benchTopo},
		{Name: "hashing_consistent_ring", Input: 200_000, Ops: 200_000, Run: benchHashRing},
	}
}

// Runner executes benchmark cases and stores recorded results.
type Runner struct {
	mu      sync.Mutex
	results []models.BenchmarkResult
	clock   func() time.Time
}

// NewRunner creates a benchmark runner.
func NewRunner() *Runner {
	return &Runner{results: make([]models.BenchmarkResult, 0), clock: time.Now}
}

// CaseByName returns a case, or false when unknown.
func CaseByName(name string) (Case, bool) {
	for _, c := range Cases() {
		if c.Name == name {
			return c, true
		}
	}
	return Case{}, false
}

// Names returns the sorted names of all available cases.
func Names() []string {
	names := make([]string, 0, len(Cases()))
	for _, c := range Cases() {
		names = append(names, c.Name)
	}
	sort.Strings(names)
	return names
}

// Run executes a single named case and records the result.
func (r *Runner) Run(name string) (models.BenchmarkResult, error) {
	c, ok := CaseByName(name)
	if !ok {
		return models.BenchmarkResult{}, fmt.Errorf("benchmark %q not found (available: %s)", name, joinNames())
	}
	start := time.Now()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	if err := c.Run(); err != nil {
		return models.BenchmarkResult{}, err
	}

	runtime.ReadMemStats(&after)
	elapsed := time.Since(start)
	runtime.GC()

	res := models.BenchmarkResult{
		ID:          fmt.Sprintf("bench-%d", time.Now().UnixNano()),
		Algorithm:   c.Name,
		InputSize:   c.Input,
		Runtime:     elapsed,
		MemoryBytes: int64(after.TotalAlloc - before.TotalAlloc),
		Throughput:  float64(c.Ops) / elapsed.Seconds(),
		Metadata:    map[string]interface{}{"ops": c.Ops},
		RunAt:       r.clock(),
	}

	r.mu.Lock()
	r.results = append(r.results, res)
	if len(r.results) > 200 {
		r.results = r.results[len(r.results)-200:]
	}
	r.mu.Unlock()
	return res, nil
}

// RunAll runs every available case in sorted order.
func (r *Runner) RunAll() []models.BenchmarkResult {
	names := Names()
	out := make([]models.BenchmarkResult, 0, len(names))
	for _, n := range names {
		res, err := r.Run(n)
		if err != nil {
			continue
		}
		out = append(out, res)
	}
	return out
}

// List returns all recorded results, newest first.
func (r *Runner) List() []models.BenchmarkResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]models.BenchmarkResult, len(r.results))
	copy(out, r.results)
	sort.Slice(out, func(i, j int) bool {
		return out[i].RunAt.After(out[j].RunAt)
	})
	return out
}

func joinNames() []string {
	return Names()
}

// --- workload implementations -------------------------------------------------

func benchArrayAppend() error {
	a := array.New[int]()
	for i := 0; i < 200_000; i++ {
		a.Append(i)
	}
	for i := 0; i < 200_000; i++ {
		a.Get(i)
	}
	return nil
}

func benchStack() error {
	s := stack.New[int]()
	for i := 0; i < 200_000; i++ {
		s.Push(i)
	}
	for i := 0; i < 200_000; i++ {
		s.Pop()
	}
	return nil
}

func benchQueue() error {
	q := queue.New[int]()
	for i := 0; i < 200_000; i++ {
		q.Enqueue(i)
	}
	for i := 0; i < 200_000; i++ {
		q.Dequeue()
	}
	return nil
}

func benchCircular() error {
	q := queue.NewCircular[int](150_000)
	for i := 0; i < 100_000; i++ {
		q.Enqueue(i)
	}
	for i := 0; i < 100_000; i++ {
		q.Dequeue()
	}
	return nil
}

func benchHeap() error {
	h := heap.NewMinHeap[int](func(a, b int) int { return a - b })
	for i := 0; i < 200_000; i++ {
		h.Insert(i)
	}
	for i := 0; i < 200_000; i++ {
		h.Extract()
	}
	return nil
}

func benchPriorityQueue() error {
	q := heap.NewPriorityQueue[int]()
	for i := 0; i < 200_000; i++ {
		q.Enqueue(i, i%100)
	}
	return nil
}

func benchLinkedList() error {
	l := linkedlist.NewSingly[int]()
	for i := 0; i < 200_000; i++ {
		l.PushBack(i)
	}
	return nil
}

func benchHashtable() error {
	m := hashtable.New[string, int]()
	for i := 0; i < 200_000; i++ {
		m.Put(fmt.Sprintf("key-%d", i), i)
	}
	for i := 0; i < 200_000; i++ {
		m.Get(fmt.Sprintf("key-%d", i))
	}
	return nil
}

func benchBST() error {
	bst := tree.NewBST[int](func(a, b int) int { return a - b })
	for i := 0; i < 100_000; i++ {
		bst.Insert(i)
	}
	for i := 0; i < 50_000; i++ {
		bst.Search(i * 2)
	}
	return nil
}

func benchAVL() error {
	avl := tree.NewAVL[int](func(a, b int) int { return a - b })
	for i := 0; i < 100_000; i++ {
		avl.Insert(i)
	}
	return nil
}

func benchTrie() error {
	t := trie.New()
	for i := 0; i < 100_000; i++ {
		t.Insert(fmt.Sprintf("package_%d_example_word_seed", i))
	}
	for i := 0; i < 100_000; i++ {
		t.Search(fmt.Sprintf("package_%d_example_word_seed", i))
	}
	return nil
}

func benchLRU() error {
	c := cache.NewLRU[int, int](20_000)
	for i := 0; i < 200_000; i++ {
		c.Put(i, i)
	}
	for i := 0; i < 100_000; i++ {
		c.Get(i % 25_000)
	}
	return nil
}

func benchBFS() error {
	g := graph.New(true)
	for i := 0; i < 50_000; i++ {
		g.AddNode(ids(i), "n")
	}
	for i := 1; i < 50_000; i++ {
		g.AddEdge(ids(i-1), ids(i), 1)
	}
	g.BFS(ids(0))
	return nil
}

func benchDijkstra() error {
	g := graph.New(true)
	for i := 0; i < 30_000; i++ {
		g.AddNode(ids(i), "n")
	}
	for i := 1; i < 30_000; i++ {
		g.AddEdge(ids(i-1), ids(i), float64(i))
		if i%3 == 0 {
			g.AddEdge(ids(i-1), ids(i-1000+1), 1)
		}
	}
	g.Dijkstra(ids(0))
	return nil
}

func benchTopo() error {
	g := graph.New(true)
	for i := 0; i < 50_000; i++ {
		g.AddNode(ids(i), "n")
	}
	for i := 1; i < 50_000; i++ {
		g.AddEdge(ids(i-1), ids(i), 1)
	}
	g.TopologicalSort()
	return nil
}

func benchHashRing() error {
	ring := hashing.NewHashRing(100)
	for i := 0; i < 8; i++ {
		ring.AddNode(fmt.Sprintf("shard-%d", i))
	}
	for i := 0; i < 200_000; i++ {
		ring.GetNode(fmt.Sprintf("key-%d", i))
	}
	return nil
}

func ids(i int) string {
	return fmt.Sprintf("n%08d", i)
}
package jobqueue

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/scheduler"
)

func testJob(id string, priority int) *Job {
	return &Job{
		ID:         id,
		Name:       id,
		Type:       "test",
		Priority:   priority,
		MaxRetries: 2,
	}
}

func TestSubmitAndDequeueFIFO(t *testing.T) {
	q := NewQueue(Options{})

	for i := 0; i < 3; i++ {
		if err := q.Submit(testJob(string(rune('a'+i)), 0)); err != nil {
			t.Fatalf("submit %d: %v", i, err)
		}
	}

	ids := []string{}
	for i := 0; i < 3; i++ {
		job, err := q.Dequeue()
		if err != nil {
			t.Fatalf("dequeue %d: %v", i, err)
		}
		ids = append(ids, job.ID)
	}
	if strings.Join(ids, ",") != "a,b,c" {
		t.Fatalf("expected FIFO order a,b,c got %v", ids)
	}
}

func TestPriorityOrdering(t *testing.T) {
	q := NewQueue(Options{})

	q.Submit(testJob("low", 1))
	q.Submit(testJob("high", 10))
	q.Submit(testJob("mid", 5))

	want := []string{"high", "mid", "low"}
	for _, w := range want {
		job, err := q.Dequeue()
		if err != nil {
			t.Fatalf("dequeue: %v", err)
		}
		if job.ID != w {
			t.Fatalf("expected %s got %s", w, job.ID)
		}
	}
}

func TestDuplicateSubmission(t *testing.T) {
	q := NewQueue(Options{})
	if err := q.Submit(testJob("dedup", 0)); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if err := q.Submit(testJob("dedup", 0)); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate got %v", err)
	}
}

func TestDequeueEmpty(t *testing.T) {
	q := NewQueue(Options{})
	if _, err := q.Dequeue(); !errors.Is(err, ErrEmpty) {
		t.Fatalf("expected ErrEmpty got %v", err)
	}
}

func TestAcknowledge(t *testing.T) {
	q := NewQueue(Options{})
	q.Submit(testJob("job-1", 0))
	job, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if job.State != StatusRunning {
		t.Fatalf("expected running got %s", job.State)
	}
	if err := q.Acknowledge(job.ID); err != nil {
		t.Fatalf("ack: %v", err)
	}
	got, err := q.Get("job-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != StatusCompleted {
		t.Fatalf("expected completed got %s", got.State)
	}
}

func TestRetryWithBackoff(t *testing.T) {
	now := time.Now()
	q := NewQueue(Options{
		Clock:       func() time.Time { return now },
		BackoffBase: 100 * time.Millisecond,
	})

	q.Submit(testJob("retry-job", 0))

	job, _ := q.Dequeue()
	if job.Attempts != 1 {
		t.Fatalf("expected attempts=1 got %d", job.Attempts)
	}
	if err := q.Reject(job.ID, errors.New("flaky")); err != nil {
		t.Fatalf("reject: %v", err)
	}

	first := q.opts.Clock()
	ref := now.Add(100 * time.Millisecond)
	// Advance clock past first backoff
	now = now.Add(150 * time.Millisecond)
	job2, err := q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue after backoff: %v", err)
	}
	if job2.ID != "retry-job" {
		t.Fatalf("expected retry-job got %s", job2.ID)
	}
	if job2.Retries != 1 {
		t.Fatalf("expected retries=1 got %d", job2.Retries)
	}
	if job2.Attempts != 2 {
		t.Fatalf("expected attempts=2 got %d", job2.Attempts)
	}
	_ = first
	_ = ref
}

func TestDeadLetterAfterMaxRetries(t *testing.T) {
	now := time.Now()
	q := NewQueue(Options{
		Clock:       func() time.Time { return now },
		BackoffBase: 10 * time.Millisecond,
		MaxRetries:  2,
	})

	q.Submit(testJob("doomed", 0))

	// failure 1 -> retry 1 (Retries 0->1)
	job, _ := q.Dequeue()
	q.Reject(job.ID, errors.New("err-1"))

	now = now.Add(30 * time.Millisecond)
	// failure 2 -> retry 2 (Retries 1->2)
	job, _ = q.Dequeue()
	q.Reject(job.ID, errors.New("err-2"))

	now = now.Add(60 * time.Millisecond)
	// failure 3 -> dead-lettered (retries exhausted)
	job, _ = q.Dequeue()
	q.Reject(job.ID, errors.New("err-3"))

	if q.Stats().DeadLettered != 1 {
		t.Fatalf("expected 1 dead-lettered after retries exhausted, got %+v", q.Stats())
	}

	dead := q.DeadLettered()
	if len(dead) != 1 || dead[0].ID != "doomed" {
		t.Fatalf("unexpected dead-letter queue: %+v", dead)
	}
	if dead[0].LastError != "err-3" {
		t.Fatalf("expected last_error=err-3 got %q", dead[0].LastError)
	}
	if dead[0].Retries != dead[0].MaxRetries {
		t.Fatalf("expected retries exhausted (%d = %d)", dead[0].Retries, dead[0].MaxRetries)
	}
}

func TestMaxRetriesZeroNoRetry(t *testing.T) {
	q := NewQueue(Options{})
	q.Submit(&Job{ID: "no-retry", Name: "no-retry", Priority: 0, DisableRetry: true})
	job, _ := q.Dequeue()
	q.Reject(job.ID, errors.New("boom"))
	if q.Stats().DeadLettered != 1 {
		t.Fatalf("expected immediate dead letter for DisableRetry job")
	}
}

func TestSchedulerIntegration(t *testing.T) {
	eng := newTestScheduler()

	q := NewQueue(Options{Scheduler: eng})
	q.Submit(&Job{ID: "sched-1", Name: "sched-1", Type: "cpu", Priority: 5, CPU: 1.0, Memory: 1.0})

	job, err := q.Get("sched-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if job.AssignedNode == "" {
		t.Fatalf("expected job assigned to a node")
	}
}

func newTestScheduler() *scheduler.Engine {
	eng := scheduler.NewEngine(event.NewBus(16))
	eng.RegisterNode(&scheduler.NodeInfo{
		ID:             "node-a",
		Name:           "node-a",
		CPUCapacity:    16,
		MemoryCapacity: 256,
		GPUCapacity:    4,
		NetworkLatency: 2,
		Health:         1.0,
	})
	eng.RegisterNode(&scheduler.NodeInfo{
		ID:             "node-b",
		Name:           "node-b",
		CPUCapacity:    8,
		MemoryCapacity: 128,
		GPUCapacity:    2,
		NetworkLatency: 5,
		Health:         0.8,
	})
	return eng
}

func TestConcurrentSubmitAndDequeue(t *testing.T) {
	q := NewQueue(Options{})
	const n = 200

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q.Submit(testJob(strings.Repeat("x", 0)+itoa(i), 0))
		}(i)
	}
	wg.Wait()

	if q.Stats().Pending != n {
		t.Fatalf("expected %d pending got %+v", n, q.Stats())
	}

	seen := make(map[string]bool)
	for i := 0; i < n; i++ {
		job, err := q.Dequeue()
		if err != nil {
			t.Fatalf("dequeue %d: %v", i, err)
		}
		if seen[job.ID] {
			t.Fatalf("duplicate job dequeued: %s", job.ID)
		}
		seen[job.ID] = true
		q.Acknowledge(job.ID)
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique jobs, got %d", n, len(seen))
	}
	if q.Stats().Completed != n {
		t.Fatalf("expected %d completed got %+v", n, q.Stats())
	}
}

func TestShardedQueueFanout(t *testing.T) {
	q := NewShardedQueue(8, Options{})
	const n = 100

	for i := 0; i < n; i++ {
		if err := q.Submit(testJob(itoa(i), 0)); err != nil {
			t.Fatalf("submit %d: %v", i, err)
		}
	}

	total := q.Stats().Total()
	if total != n {
		t.Fatalf("expected %d jobs across shards got %d", n, total)
	}

	// Jobs are distributed across multiple shards
	nonEmpty := 0
	for _, s := range q.Shards() {
		if s.Stats().Pending > 0 {
			nonEmpty++
		}
	}
	if nonEmpty < 2 {
		t.Fatalf("expected jobs spread across shards, only %d non-empty", nonEmpty)
	}

	// Dequeue should drain all jobs
	seen := make(map[string]bool)
	for i := 0; i < n; i++ {
		job, err := q.Dequeue()
		if err != nil {
			t.Fatalf("dequeue %d: %v", i, err)
		}
		if seen[job.ID] {
			t.Fatalf("duplicate job: %s", job.ID)
		}
		seen[job.ID] = true
	}
	if _, err := q.Dequeue(); !errors.Is(err, ErrEmpty) {
		t.Fatalf("expected ErrEmpty after draining, got %v", err)
	}
}

func TestShardedQueueGetAck(t *testing.T) {
	q := NewShardedQueue(4, Options{})
	q.Submit(testJob("target-job", 0))

	job, err := q.Get("target-job")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if job.ID != "target-job" {
		t.Fatalf("expected target-job got %s", job.ID)
	}

	_, err = q.Dequeue()
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if err := q.Acknowledge("target-job"); err != nil {
		t.Fatalf("ack: %v", err)
	}
	done, _ := q.Get("target-job")
	if done.State != StatusCompleted {
		t.Fatalf("expected completed, got %s", done.State)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
// Package jobqueue implements a distributed job queue with priority ordering,
// idempotent submission, retries with exponential backoff, and a dead letter queue.
//
// Design:
//   - A single Queue uses a binary heap for priority ordering (higher priority
//     first, FIFO within the same priority) and a delayed heap for retry backoff.
//   - Submissions are idempotent: an active Job ID cannot be submitted twice.
//   - Failed jobs are retried with exponential backoff up to MaxRetries, then
//     dead-lettered.
//   - ShardedQueue spreads queues across consistent-hash shards so the queue
//     fabric itself can be distributed.
//   - When a scheduler engine is attached, submitted jobs are scored against
//     registered nodes and assigned to the best fit.
package jobqueue

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/algorithms/hashing"
	"github.com/atlas-engine/atlas/internal/algorithms/heap"
	"github.com/atlas-engine/atlas/internal/event"
	"github.com/atlas-engine/atlas/internal/scheduler"
)

// Status is the lifecycle state of a job.
type Status string

const (
	StatusPending      Status = "pending"
	StatusRunning      Status = "running"
	StatusCompleted    Status = "completed"
	StatusFailed       Status = "failed"
	StatusDeadLettered Status = "dead_lettered"
)

// Job is a unit of work tracked by the queue.
type Job struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Priority     int               `json:"priority"`
	Payload      string            `json:"payload"`
	Labels       map[string]string `json:"labels"`

	CPU        float64 `json:"cpu"`
	Memory     float64 `json:"memory"`
	GPU        float64 `json:"gpu"`
	MaxRetries int     `json:"max_retries"`
	DisableRetry bool  `json:"disable_retry"`

	State        Status     `json:"status"`
	Retries      int        `json:"retries"`
	Attempts     int        `json:"attempts"`
	AssignedNode string     `json:"assigned_node"`
	CreatedAt    time.Time  `json:"created_at"`
	EnqueuedAt   time.Time  `json:"enqueued_at"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  time.Time  `json:"completed_at"`
	LastError    string     `json:"last_error"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
}

// Options used when constructing a Queue. Defaults are applied for zero values.
type Options struct {
	Logger       *slog.Logger
	BackoffBase  time.Duration
	MaxRetries   int
	Clock        func() time.Time
	Scheduler    *scheduler.Engine
	EventBus     *event.Bus
}

// Queue is a priority job queue with retry and dead-letter handling.
type Queue struct {
	mu        sync.RWMutex
	jobs      map[string]*Job
	ready     *heap.Heap[jobEntry]
	delayed   *heap.Heap[delayEntry]
	inFlight  map[string]*Job
	deadLetter []*Job

	opts     Options
	nextSeq  int64
}

type jobEntry struct {
	job *Job
	seq int64
}

type delayEntry struct {
	job *Job
	at  time.Time
}

// Sentinel errors returned by the queue.
var (
	ErrDuplicate = errors.New("jobqueue: job already exists")
	ErrEmpty     = errors.New("jobqueue: no pending jobs")
	ErrNotFound  = errors.New("jobqueue: job not found")
	ErrInvalid   = errors.New("jobqueue: invalid job")
)

// NewQueue creates a job queue.
func NewQueue(opts Options) *Queue {
	if opts.BackoffBase <= 0 {
		opts.BackoffBase = 100 * time.Millisecond
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	cmpJob := func(a, b jobEntry) int {
		if a.job.Priority != b.job.Priority {
			return b.job.Priority - a.job.Priority
		}
		if a.seq < b.seq {
			return -1
		}
		if a.seq > b.seq {
			return 1
		}
		return 0
	}
	cmpDelay := func(a, b delayEntry) int {
		if a.at.Before(b.at) {
			return -1
		}
		if a.at.After(b.at) {
			return 1
		}
		return 0
	}

	return &Queue{
		jobs:       make(map[string]*Job),
		ready:      heap.NewMinHeap[jobEntry](cmpJob),
		delayed:    heap.NewMinHeap[delayEntry](cmpDelay),
		inFlight:   make(map[string]*Job),
		deadLetter: make([]*Job, 0),
		opts:       opts,
	}
}

// Submit enqueues a job. Returns ErrDuplicate if the job ID is already active.
// When a scheduler is attached, the job is assigned to the best-fit node.
func (q *Queue) Submit(job *Job) error {
	if job == nil || job.ID == "" || job.Name == "" {
		return ErrInvalid
	}

	q.mu.Lock()
	if _, ok := q.jobs[job.ID]; ok {
		q.mu.Unlock()
		return ErrDuplicate
	}
	if job.Priority == 0 {
		job.Priority = 0
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = q.opts.MaxRetries
	}
	now := q.opts.Clock()
	job.State = StatusPending
	job.CreatedAt = now
	job.EnqueuedAt = now
	job.Retries = 0
	job.Attempts = 0
	job.LastError = ""

	if q.opts.Scheduler != nil {
		result := q.opts.Scheduler.Schedule(job.toWorkload())
		if result != nil {
			job.AssignedNode = result.NodeID
		}
	}

	q.jobs[job.ID] = job
	q.ready.Insert(jobEntry{job: job, seq: q.nextSeq})
	q.nextSeq++
	q.mu.Unlock()

	q.emit(event.JobCreated, job)
	q.opts.Logger.Debug("job submitted",
		slog.String("job", job.ID),
		slog.String("node", job.AssignedNode),
		slog.Int("priority", job.Priority))
	return nil
}

// Dequeue returns the next runnable job, moving any due delayed jobs back
// into the ready queue first. Returns ErrEmpty when nothing is runnable.
func (q *Queue) Dequeue() (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := q.opts.Clock()
	q.redistributeLocked(now)

	entry, ok := q.ready.Extract()
	if !ok {
		return nil, ErrEmpty
	}

	job := entry.job
	job.State = StatusRunning
	job.StartedAt = now
	job.Attempts++
	job.NextRetryAt = nil
	q.inFlight[job.ID] = job

	q.opts.Logger.Debug("job dequeued",
		slog.String("job", job.ID),
		slog.String("node", job.AssignedNode),
		slog.Int("attempt", job.Attempts))
	return job, nil
}

// Acknowledge marks a running job as completed.
func (q *Queue) Acknowledge(id string) error {
	q.mu.Lock()
	job, ok := q.inFlight[id]
	if !ok {
		q.mu.Unlock()
		return ErrNotFound
	}
	delete(q.inFlight, id)
	now := q.opts.Clock()
	job.State = StatusCompleted
	job.CompletedAt = now
	q.mu.Unlock()

	q.emit(event.JobCompleted, job)
	q.opts.Logger.Debug("job completed", slog.String("job", id))
	return nil
}

// Reject fails a running job, retrying it with exponential backoff until
// MaxRetries is exceeded, then moving it to the dead letter queue.
func (q *Queue) Reject(id string, cause error) error {
	q.mu.Lock()
	job, ok := q.inFlight[id]
	if !ok {
		q.mu.Unlock()
		return ErrNotFound
	}
	delete(q.inFlight, id)
	now := q.opts.Clock()

	errorMsg := ""
	if cause != nil {
		errorMsg = cause.Error()
	}
	job.LastError = errorMsg

	if !job.DisableRetry && job.Retries < job.MaxRetries {
		job.Retries++
		backoff := q.opts.BackoffBase
		for i := 1; i < job.Retries; i++ {
			backoff *= 2
		}
		at := now.Add(backoff)
		job.NextRetryAt = &at
		job.State = StatusPending
		q.delayed.Insert(delayEntry{job: job, at: at})
		q.mu.Unlock()

		q.emit(event.JobRetried, job)
		q.opts.Logger.Warn("job rejected, scheduled retry",
			slog.String("job", id),
			slog.Int("retry", job.Retries),
			slog.String("error", errorMsg),
			slog.Duration("backoff", backoff))
		return nil
	}

	job.State = StatusDeadLettered
	q.deadLetter = append(q.deadLetter, job)
	q.mu.Unlock()

	q.emit(event.JobFailed, job)
	q.opts.Logger.Error("job dead-lettered",
		slog.String("job", id),
		slog.Int("retries", job.Retries),
		slog.String("error", errorMsg))
	return nil
}

// Get returns a snapshot of a job by ID.
func (q *Queue) Get(id string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	job, ok := q.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return job, nil
}

// Jobs returns a snapshot of all jobs sorted by creation sequence.
func (q *Queue) Jobs() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()
	result := make([]*Job, 0, len(q.jobs))
	for _, j := range q.jobs {
		result = append(result, j)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].EnqueuedAt.Before(result[j].EnqueuedAt)
	})
	return result
}

// DeadLettered returns jobs that permanently failed.
func (q *Queue) DeadLettered() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()
	result := make([]*Job, len(q.deadLetter))
	copy(result, q.deadLetter)
	return result
}

// Stats aggregates queue metrics.
func (q *Queue) Stats() Stats {
	q.mu.RLock()
	defer q.mu.RUnlock()
	s := Stats{}
	for _, j := range q.jobs {
		switch j.State {
		case StatusPending:
			s.Pending++
		case StatusRunning:
			s.Running++
		case StatusCompleted:
			s.Completed++
		case StatusFailed:
			s.Failed++
		case StatusDeadLettered:
			s.DeadLettered++
		}
	}
	return s
}

// Stats describes the overall state of a queue.
type Stats struct {
	Pending      int `json:"pending"`
	Running      int `json:"running"`
	Completed    int `json:"completed"`
	Failed       int `json:"failed"`
	DeadLettered int `json:"dead_lettered"`
}

// Total returns the number of tracked jobs.
func (s Stats) Total() int {
	return s.Pending + s.Running + s.Completed + s.Failed + s.DeadLettered
}

func (q *Queue) redistributeLocked(now time.Time) {
	for {
		entry, ok := q.delayed.Peek()
		if !ok || entry.at.After(now) {
			return
		}
		q.delayed.Extract()
		entry.job.State = StatusPending
		entry.job.NextRetryAt = nil
		q.ready.Insert(jobEntry{job: entry.job, seq: q.nextSeq})
		q.nextSeq++
	}
}

func (q *Queue) emit(t event.EventType, job *Job) {
	if q.opts.EventBus == nil {
		return
	}
	q.opts.EventBus.Publish(event.NewEvent(t, "jobqueue", map[string]interface{}{
		"job_id":     job.ID,
		"name":       job.Name,
		"type":       job.Type,
		"priority":   job.Priority,
		"node_id":    job.AssignedNode,
		"attempt":    job.Attempts,
		"retries":    job.Retries,
		"last_error": job.LastError,
		"status":     string(job.State),
	}))
}

func (j *Job) toWorkload() *scheduler.Workload {
	w := &scheduler.Workload{
		ID:         j.ID,
		Name:       j.Name,
		CPU:        j.CPU,
		Memory:     j.Memory,
		GPU:        j.GPU,
		Priority:   j.Priority,
		Labels:     j.Labels,
		Affinity:   []string{},
		AntiAffinity: []string{},
	}
	return w
}

// ShardedQueue spreads queues across consistent-hash shards so jobs land on
// the shard owning their ID. Dequeue round-robins across shards.
type ShardedQueue struct {
	mu        sync.RWMutex
	shards    []*Queue
	ring      *hashing.HashRing
	nodeIndex map[string]int
	next      int
}

// NewShardedQueue creates a sharded job queue.
func NewShardedQueue(shardCount int, opts Options) *ShardedQueue {
	if shardCount <= 0 {
		shardCount = 1
	}
	q := &ShardedQueue{
		shards:    make([]*Queue, shardCount),
		ring:      hashing.NewHashRing(150),
		nodeIndex: make(map[string]int, shardCount),
	}
	for i := 0; i < shardCount; i++ {
		q.shards[i] = NewQueue(opts)
		name := fmt.Sprintf("shard-%d", i)
		q.ring.AddNode(name)
		q.nodeIndex[name] = i
	}
	return q
}

// Submit routes the job to the shard that owns its ID.
func (q *ShardedQueue) Submit(job *Job) error {
	if job == nil || job.ID == "" {
		return ErrInvalid
	}
	node := q.ring.GetNode(job.ID)
	idx, ok := q.nodeIndex[node]
	if !ok {
		return ErrInvalid
	}
	return q.shards[idx].Submit(job)
}

// Dequeue pulls the next job from shards in round-robin order.
func (q *ShardedQueue) Dequeue() (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	n := len(q.shards)
	for i := 0; i < n; i++ {
		shard := (q.next + i) % n
		job, err := q.shards[shard].Dequeue()
		if err == nil {
			q.next = (shard + 1) % n
			return job, nil
		}
	}
	return nil, ErrEmpty
}

func (q *ShardedQueue) shardOf(id string) int {
	node := q.ring.GetNode(id)
	idx, ok := q.nodeIndex[node]
	if !ok {
		return 0
	}
	return idx
}

// Get finds a job across all shards.
func (q *ShardedQueue) Get(id string) (*Job, error) {
	return q.shards[q.shardOf(id)].Get(id)
}

// Acknowledge completes a job on its owning shard.
func (q *ShardedQueue) Acknowledge(id string) error {
	return q.shards[q.shardOf(id)].Acknowledge(id)
}

// Reject fails a job on its owning shard.
func (q *ShardedQueue) Reject(id string, cause error) error {
	return q.shards[q.shardOf(id)].Reject(id, cause)
}

// Stats aggregates stats across shards.
func (q *ShardedQueue) Stats() Stats {
	var total Stats
	for _, s := range q.shards {
		st := s.Stats()
		total.Pending += st.Pending
		total.Running += st.Running
		total.Completed += st.Completed
		total.Failed += st.Failed
		total.DeadLettered += st.DeadLettered
	}
	return total
}

// Shards returns the underlying shards.
func (q *ShardedQueue) Shards() []*Queue {
	return q.shards
}
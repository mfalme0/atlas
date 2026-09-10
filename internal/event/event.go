package event

import (
	"fmt"
	"sync"
	"time"
)

type EventType string

const (
	NodeDiscovered           EventType = "node.discovered"
	NodeRemoved              EventType = "node.removed"
	NodeFailed               EventType = "node.failed"
	NodeRecovered            EventType = "node.recovered"
	ServiceStarted           EventType = "service.started"
	ServiceStopped           EventType = "service.stopped"
	JobCreated               EventType = "job.created"
	JobAssigned              EventType = "job.assigned"
	JobCompleted             EventType = "job.completed"
	JobFailed                EventType = "job.failed"
	JobRetried               EventType = "job.retried"
	LeaderElected            EventType = "leader.elected"
	LeaderFailed             EventType = "leader.failed"
	AnomalyDetected          EventType = "anomaly.detected"
	IncidentCreated          EventType = "incident.created"
	ChaosExperimentStarted   EventType = "chaos.started"
	ChaosExperimentCompleted EventType = "chaos.completed"
	TopologyChanged          EventType = "topology.changed"
	WorkerRegistered         EventType = "worker.registered"
	WorkerFailed             EventType = "worker.failed"
)

type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
}

type EventHandler func(Event)

type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]EventHandler
	buffer   chan Event
	capacity int
}

func NewBus(capacity int) *Bus {
	if capacity <= 0 {
		capacity = 1000
	}
	return &Bus{
		handlers: make(map[EventType][]EventHandler),
		buffer:   make(chan Event, capacity),
		capacity: capacity,
	}
}

func (b *Bus) Subscribe(eventType EventType, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *Bus) Publish(e Event) {
	select {
	case b.buffer <- e:
	default:
	}
}

func (b *Bus) Start() {
	go func() {
		for e := range b.buffer {
			b.mu.RLock()
			handlers := b.handlers[e.Type]
			b.mu.RUnlock()
			for _, h := range handlers {
				go h(e)
			}
		}
	}()
}

func (b *Bus) Close() {
	close(b.buffer)
}

func NewEvent(eventType EventType, source string, data map[string]interface{}) Event {
	return Event{
		ID:        generateID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Source:    source,
		Data:      data,
	}
}

var idCounter uint64
var idMu sync.Mutex

func generateID() string {
	idMu.Lock()
	idCounter++
	id := idCounter
	idMu.Unlock()
	return fmt.Sprintf("evt-%d", id)
}

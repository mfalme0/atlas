// Package metrics provides a small in-process registry of counters and
// gauges, a Prometheus text-format renderer, and a recorder that derives
// counters from the event bus.
//
// The registry is intentionally dependency-free: any engine can Inc/Gauge
// without importing Prometheus client code. Rendering produces one line per
// metric so Grafana/ Prometheus scrapers can consume the endpoint directly.
package metrics

import (
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
)

// Registry stores counters and gauges under quoted keys.
type Registry struct {
	mu       sync.RWMutex
	counters map[string]int64
	gauges   map[string]float64
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

// Inc increments a counter by delta.
func (r *Registry) Inc(name string, delta int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name] += delta
}

// Counter returns the current value of a counter.
func (r *Registry) Counter(name string) int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.counters[name]
}

// Gauge sets a gauge to value.
func (r *Registry) Gauge(name string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = value
}

// GaugeValue returns the current gauge value.
func (r *Registry) GaugeValue(name string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.gauges[name]
}

// CounterNames returns sorted counter names.
func (r *Registry) CounterNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.counters))
	for name := range r.counters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GaugeNames returns sorted gauge names.
func (r *Registry) GaugeNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.gauges))
	for name := range r.gauges {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RenderPrometheus writes the registry in Prometheus text exposition format
// (version 0.0.4): a # TYPE line followed by the value line per metric.
func (r *Registry) RenderPrometheus(w io.Writer) error {
	for _, name := range r.CounterNames() {
		v := r.Counter(name)
		if _, err := fmt.Fprintf(w, "# TYPE %s counter\n%s %d\n", sanitize(name), sanitize(name), v); err != nil {
			return err
		}
	}
	for _, name := range r.GaugeNames() {
		v := r.GaugeValue(name)
		if _, err := fmt.Fprintf(w, "# TYPE %s gauge\n%s %v\n", sanitize(name), sanitize(name), v); err != nil {
			return err
		}
	}
	return nil
}

func sanitize(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == ':':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// BusRecorder subscribes to the event bus and mirrors every event into a
// counter of the form atlas_events_{event_type}_total, plus an aggregate
// atlas_events_total.
type BusRecorder struct {
	reg     *Registry
	types   []event.EventType
}

// NewBusRecorder subscribes to the bus and starts distributing events.
func NewBusRecorder(bus *event.Bus, reg *Registry) *BusRecorder {
	rec := &BusRecorder{reg: reg}
	if bus == nil {
		return rec
	}
	for _, t := range eventTypesOfInterest {
		bus.Subscribe(t, rec.onEvent)
	}
	return rec
}

func (rec *BusRecorder) onEvent(e event.Event) {
	rec.reg.Inc("atlas_events_total", 1)
	rec.reg.Inc("atlas_events_"+string(sanitize(string(e.Type)))+"_total", 1)
}

var eventTypesOfInterest = []event.EventType{
	event.NodeDiscovered,
	event.NodeRemoved,
	event.NodeFailed,
	event.NodeRecovered,
	event.JobCreated,
	event.JobAssigned,
	event.JobCompleted,
	event.JobFailed,
	event.JobRetried,
	event.LeaderElected,
	event.AnomalyDetected,
	event.IncidentCreated,
	event.ChaosExperimentStarted,
	event.ChaosExperimentCompleted,
	event.TopologyChanged,
}

// RunRuntimeCollector updates process-level gauges every interval: goroutine
// count, allocated heap bytes, and process uptime. It stops when stop closes.
func RunRuntimeCollector(reg *Registry, interval time.Duration, stop <-chan struct{}) {
	start := time.Now()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	updateRuntime(reg, start)
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			updateRuntime(reg, start)
		}
	}
}

func updateRuntime(reg *Registry, start time.Time) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	reg.Gauge("atlas_runtime_goroutines", float64(runtime.NumGoroutine()))
	reg.Gauge("atlas_runtime_mem_alloc_bytes", float64(m.Alloc))
	reg.Gauge("atlas_process_uptime_seconds", time.Since(start).Seconds())
}
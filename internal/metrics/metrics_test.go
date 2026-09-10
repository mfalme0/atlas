package metrics

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
)

func TestCountersAndGauges(t *testing.T) {
	reg := NewRegistry()
	reg.Inc("requests_total", 3)
	reg.Inc("requests_total", 2)
	reg.Gauge("cpu_usage", 0.42)

	if reg.Counter("requests_total") != 5 {
		t.Fatalf("expected 5, got %d", reg.Counter("requests_total"))
	}
	if reg.GaugeValue("cpu_usage") != 0.42 {
		t.Fatalf("expected 0.42, got %v", reg.GaugeValue("cpu_usage"))
	}
}

func TestRenderPrometheus(t *testing.T) {
	reg := NewRegistry()
	reg.Inc("atlas_events_total", 7)
	reg.Gauge("atlas_health_score", 0.9)

	var buf bytes.Buffer
	if err := reg.RenderPrometheus(&buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"# TYPE atlas_events_total counter",
		"atlas_events_total 7",
		"# TYPE atlas_health_score gauge",
		"atlas_health_score 0.9",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("atlas.events/node*x"); got != "atlas_events_node_x" {
		t.Fatalf("got %q", got)
	}
}

func TestBusRecorderCounts(t *testing.T) {
	bus := event.NewBus(16)
	bus.Start()
	reg := NewRegistry()
	NewBusRecorder(bus, reg)

	bus.Publish(event.NewEvent(event.NodeFailed, "test", map[string]interface{}{}))
	bus.Publish(event.NewEvent(event.AnomalyDetected, "test", map[string]interface{}{}))
	bus.Publish(event.NewEvent(event.AnomalyDetected, "test", map[string]interface{}{}))
	time.Sleep(50 * time.Millisecond)

	if reg.Counter("atlas_events_total") != 3 {
		t.Fatalf("expected 3 total events, got %d", reg.Counter("atlas_events_total"))
	}
	if reg.Counter("atlas_events_node_failed_total") != 1 {
		t.Fatalf("expected 1 node_failed event, got %d", reg.Counter("atlas_events_node_failed_total"))
	}
	if reg.Counter("atlas_events_anomaly_detected_total") != 2 {
		t.Fatalf("expected 2 anomaly events, got %d", reg.Counter("atlas_events_anomaly_detected_total"))
	}
}
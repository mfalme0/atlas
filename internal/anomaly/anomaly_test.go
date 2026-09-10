package anomaly

import (
	"testing"
	"time"
)

func TestNoDetectionWhenCold(t *testing.T) {
	d := NewDetector(Options{})

	for i := 0; i < d.opts.MinSamples-1; i++ {
		if a := d.Record("node-a", "cpu", 50); a != nil {
			t.Fatalf("expected no anomaly before warm-up, got %+v", a)
		}
	}
	if len(d.History()) != 0 {
		t.Fatalf("expected empty history, got %d", len(d.History()))
	}
}

func TestZScoreSpikeDetection(t *testing.T) {
	d := NewDetector(Options{ZScoreThreshold: 3.0})

	for i := 0; i < 20; i++ {
		d.Record("node-a", "cpu", 40)
	}

	var spike *Anomaly
	for i := 0; i < 3; i++ {
		d.Record("node-a", "cpu", 40)
		spike = d.Record("node-a", "cpu", 95)
		if spike != nil {
			break
		}
	}
	if spike == nil {
		t.Fatalf("expected spike to be detected")
	}
	if spike.Metric != "cpu" || spike.NodeID != "node-a" {
		t.Fatalf("unexpected anomaly: %+v", spike)
	}
	if spike.Severity != SeverityWarning {
		t.Fatalf("expected warning severity, got %s", spike.Severity)
	}
	if spike.ZScore <= 0 {
		t.Fatalf("expected positive z-score, got %v", spike.ZScore)
	}
}

func TestSteadyStateNoDetection(t *testing.T) {
	d := NewDetector(Options{})

	for i := 0; i < 100; i++ {
		if a := d.Record("node-a", "mem", 60); a != nil {
			t.Fatalf("steady state should not detect, got %+v", a)
		}
	}
}

func TestRuleBreach(t *testing.T) {
	d := NewDetector(Options{})
	d.AddRule(Rule{Metric: "cpu", CriticalHigh: 90})

	// Fill baseline at a low level so no z-score trip.
	for i := 0; i < 20; i++ {
		d.Record("node-a", "cpu", 20)
	}

	a := d.Record("node-a", "cpu", 97)
	if a == nil {
		t.Fatalf("expected rule breach detected")
	}
	if a.RuleBreach != "critical_high" {
		t.Fatalf("expected critical_high breach, got %q", a.RuleBreach)
	}
	if a.Severity != SeverityCritical {
		t.Fatalf("expected critical severity, got %s", a.Severity)
	}

	low := d.Record("node-a", "cpu", 0.5)
	if low != nil {
		t.Fatalf("expect non-breach sample not detected, got %+v", low)
	}
}

func TestSummaryAndHistoryOrder(t *testing.T) {
	now := time.Now()
	d := NewDetector(Options{
		Clock: func() time.Time { return now },
	})

	d.AddRule(Rule{Metric: "cpu", CriticalHigh: 90})
	for i := 0; i < 20; i++ {
		d.Record("node-a", "cpu", 20)
		d.Record("node-b", "mem", 500)
	}
	d.Record("node-a", "cpu", 95)
	d.Record("node-b", "mem", 8000)

	if got := d.Summary()["cpu"]; got != 1 {
		t.Fatalf("expected 1 cpu anomaly, got %d", got)
	}
	if got := d.Summary()["mem"]; got != 1 {
		t.Fatalf("expected 1 mem anomaly, got %d", got)
	}
	if len(d.History()) != 2 {
		t.Fatalf("expected 2 in history, got %d", len(d.History()))
	}
}

func TestPerMetricIndependence(t *testing.T) {
	d := NewDetector(Options{})
	for i := 0; i < 30; i++ {
		d.Record("node-a", "cpu", 10)
		d.Record("node-a", "mem", 500)
	}
	// Big mem spike must not be influenced by flat cpu baseline.
	if a := d.Record("node-a", "mem", 9000); a == nil {
		t.Fatalf("expected mem spike detected independently")
	}
}
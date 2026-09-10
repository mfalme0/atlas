// Package anomaly provides statistical anomaly detection over metric streams.
//
// Detection combines two signals:
//   - Z-score analysis over a rolling window: a sample is anomalous when it
//     deviates from the recent mean by more than ZScoreThreshold standard
//     deviations (requires MinSamples samples to be warm).
//   - Absolute rule thresholds: a sample breaches a configured critical high
//     or low bound regardless of the z-score.
//
// Detected anomalies carry an severity label, the observed value, the rolling
// baseline, and a deviation score. Each detection emits an anomaly.detected
// event on the event bus.
package anomaly

import (
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/atlas-engine/atlas/internal/event"
)

// Severity classifies how far a sample deviates from normal.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Rule defines absolute thresholds for a metric. Zero values are ignored.
type Rule struct {
	Metric         string  `json:"metric"`
	CriticalHigh   float64 `json:"critical_high,omitempty"`
	CriticalLow    float64 `json:"critical_low,omitempty"`
}

// Options configure a Detector.
type Options struct {
	WindowSize    int           // rolling samples kept per metric
	MinSamples    int           // samples needed before z-score detection
	ZScoreThreshold float64     // |z| above which a sample is anomalous
	Bus           *event.Bus
	Logger        *slog.Logger
	Clock         func() time.Time
}

// Anomaly is a single detection event.
type Anomaly struct {
	ID         string     `json:"id"`
	NodeID     string     `json:"node_id"`
	Metric     string     `json:"metric"`
	Value      float64    `json:"value"`
	Mean       float64    `json:"mean"`
	StdDev     float64    `json:"stddev"`
	ZScore     float64    `json:"z_score"`
	RuleBreach string     `json:"rule_breach,omitempty"`
	Severity   Severity   `json:"severity"`
	Timestamp  time.Time  `json:"timestamp"`
}

// Detector watches metric streams and flags anomalies.
type Detector struct {
	mu      sync.RWMutex
	opts    Options
	windows map[string]*window
	rules   map[string]Rule
	history []*Anomaly
	nextID  int
}

type window struct {
	samples []float64
}

func (w *window) push(v float64, capacity int) {
	w.samples = append(w.samples, v)
	if len(w.samples) > capacity {
		w.samples = w.samples[len(w.samples)-capacity:]
	}
}

// NewDetector creates a detector. Defaults: window 30, min samples 10,
// z-score threshold 3.0.
func NewDetector(opts Options) *Detector {
	if opts.WindowSize <= 0 {
		opts.WindowSize = 30
	}
	if opts.MinSamples <= 0 {
		opts.MinSamples = 10
	}
	if opts.ZScoreThreshold <= 0 {
		opts.ZScoreThreshold = 3.0
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Detector{
		opts:    opts,
		windows: make(map[string]*window),
		rules:   make(map[string]Rule),
		history: make([]*Anomaly, 0),
	}
}

// AddRule registers an absolute threshold rule for a metric (overwrites).
func (d *Detector) AddRule(rule Rule) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rules[rule.Metric] = rule
}

// Record ingests a sample and returns a detection if the sample is anomalous.
func (d *Detector) Record(nodeID, metric string, value float64) *Anomaly {
	d.mu.Lock()

	key := nodeID + "\x00" + metric
	w := d.windows[key]
	if w == nil {
		w = &window{}
		d.windows[key] = w
	}
	oldest := firstValue(w.samples)
	w.push(value, d.opts.WindowSize)

	mean := meanOf(w.samples)
	std := stdOf(w.samples)

	var z float64
	if std > 0 {
		z = (value - mean) / std
	}

	sev := SeverityInfo
	reason := ""
	if rule, ok := d.rules[metric]; ok {
		if rule.CriticalHigh > 0 && value > rule.CriticalHigh {
			sev, reason = SeverityCritical, "critical_high"
		} else if rule.CriticalLow > 0 && value < rule.CriticalLow {
			sev, reason = SeverityCritical, "critical_low"
		}
	}

	zAnomalous := len(w.samples) >= d.opts.MinSamples && math.Abs(z) > d.opts.ZScoreThreshold
	if zAnomalous {
		if sev != SeverityCritical {
			sev = SeverityWarning
		}
		if reason == "" {
			reason = "z_score"
		}
	}

	if !zAnomalous && reason == "" {
		d.mu.Unlock()
		return nil
	}

	d.nextID++
	a := &Anomaly{
		ID:         anomalyID(d.nextID),
		NodeID:     nodeID,
		Metric:     metric,
		Value:      value,
		Mean:       mean,
		StdDev:     std,
		ZScore:     z,
		RuleBreach: reason,
		Severity:   sev,
		Timestamp:  d.opts.Clock(),
	}
	d.history = append(d.history, a)
	if len(d.history) > d.opts.WindowSize*4 {
		d.history = d.history[len(d.history)-d.opts.WindowSize*4:]
	}
	d.mu.Unlock()

	_ = oldest
	d.opts.Logger.Warn("anomaly detected",
		slog.String("node", nodeID),
		slog.String("metric", metric),
		slog.Float64("value", value),
		slog.Float64("mean", mean),
		slog.Float64("z_score", z),
		slog.String("severity", string(sev)),
		slog.String("breach", reason))
	d.emit(a)
	return a
}

// History returns recent anomalies, newest first.
func (d *Detector) History() []*Anomaly {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]*Anomaly, len(d.history))
	copy(result, d.history)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})
	return result
}

// Summary aggregates detections by metric.
func (d *Detector) Summary() map[string]int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make(map[string]int)
	for _, a := range d.history {
		out[a.Metric]++
	}
	return out
}

func (d *Detector) emit(a *Anomaly) {
	if d.opts.Bus == nil {
		return
	}
	d.opts.Bus.Publish(event.NewEvent(event.AnomalyDetected, "anomaly", map[string]interface{}{
		"anomaly_id": a.ID,
		"node_id":    a.NodeID,
		"metric":     a.Metric,
		"value":      a.Value,
		"mean":       a.Mean,
		"z_score":    a.ZScore,
		"severity":   string(a.Severity),
		"breach":     a.RuleBreach,
	}))
}

func firstValue(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	return samples[0]
}

func meanOf(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range samples {
		sum += v
	}
	return sum / float64(len(samples))
}

func stdOf(samples []float64) float64 {
	n := len(samples)
	if n < 2 {
		return 0
	}
	mean := meanOf(samples)
	var ss float64
	for _, v := range samples {
		diff := v - mean
		ss += diff * diff
	}
	return math.Sqrt(ss / float64(n-1))
}

func anomalyID(n int) string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 6)
	for i := range b {
		b[i] = digits[n%len(digits)]
		n /= len(digits)
	}
	return "an-" + string(b)
}
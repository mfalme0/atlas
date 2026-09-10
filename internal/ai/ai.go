// Package ai provides deterministic, advisory incident analysis.
//
// Safety boundary (ADR-007): the Advisor is strictly read-only. It consumes
// incident signals plus a read-only topology snapshot and produces a root
// cause hypothesis, observations, and remediation *recommendations*. No code
// path in this package mutates nodes, jobs, or cluster state, and every
// recommendation is advisory pending human approval.
package ai

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atlas-engine/atlas/internal/incident"
	"github.com/atlas-engine/atlas/internal/models"
)

// Snapshot is the read-only view of topology available to the advisor.
type Snapshot interface {
	GetNode(id string) (*models.Node, bool)
	HealthScore() float64
	Bottlenecks() []string
}

// Archetype classifies the shape of an incident.
type Archetype string

const (
	ArchetypeSingleFailure     Archetype = "single_component_failure"
	ArchetypeCascade           Archetype = "dependency_cascade"
	ArchetypeCapacityBottleneck Archetype = "capacity_bottleneck"
	ArchetypeClusterPressure   Archetype = "cluster_wide_pressure"
	ArchetypeChaosInjection    Archetype = "deliberate_failure_injection"
	ArchetypeLatencyDegradation Archetype = "latency_degradation"
	ArchetypeIndeterminate     Archetype = "indeterminate"
)

// Recommendation is an advisory remediation that requires a human to apply.
type Recommendation struct {
	Action           string `json:"action"`
	Target           string `json:"target,omitempty"`
	RequiresApproval bool   `json:"requires_approval"`
	Note             string `json:"note,omitempty"`
}

// Analysis is the complete advisory result for one incident.
type Analysis struct {
	Archetype       Archetype         `json:"archetype"`
	RootCause       string            `json:"root_cause"`
	Confidence      float64           `json:"confidence"`
	Observations    []string          `json:"observations"`
	Recommendations []Recommendation  `json:"recommendations"`
	GeneratedAt     time.Time         `json:"generated_at"`
}

// Advisor classifies incidents from signals and topology state.
type Advisor struct {
	clusterThreshold int
	clock            func() time.Time
}

// NewAdvisor creates an advisor. clusterThreshold is the number of distinct
// affected nodes that switches classification to a cluster-wide archetype.
func NewAdvisor(clusterThreshold int) *Advisor {
	if clusterThreshold <= 0 {
		clusterThreshold = 3
	}
	return &Advisor{clusterThreshold: clusterThreshold, clock: time.Now}
}

// Analyze produces an advisory analysis. It never mutates any input.
func (a *Advisor) Analyze(v *incident.View, signals []incident.Signal, snap Snapshot) *Analysis {
	analysis := &Analysis{
		Archetype:   ArchetypeIndeterminate,
		RootCause:   "No single root cause could be determined from available signals.",
		Confidence:  0.2,
		GeneratedAt: a.clock(),
	}
	if v == nil {
		return analysis
	}

	affected := v.AffectedNodes
	chaosActive, chaosTargets := hasChaos(signals)
	anomalyMetrics, anomalyNodes := metricSpread(signals)
	failedCount, degradedCount := stateCounts(affected, snap)

	analysis.Observations = a.observations(affected, snap)

	if chaosActive {
		analysis.Archetype = ArchetypeChaosInjection
		analysis.RootCause = "An active chaos experiment is the probable trigger; observed failures align with injected failure on " +
			joinTargets(chaosTargets) + "."
		analysis.Confidence = 0.85
		analysis.Recommendations = []Recommendation{
			recommend("Review the running chaos experiment and its stated duration before acting", "", "Correlate incident timestamps with experiment start"),
			recommend("Confirm whether the failure is the intended blast radius", strings.Join(chaosTargets, ","), "Do not remediate injected failures as real faults"),
		}
		return analysis
	}

	switch {
	case failedCount > 1 && multipleAffected(affected):
		analysis.Archetype = ArchetypeCascade
		analysis.RootCause = fmt.Sprintf("%d components are failed with dependents affected, indicating a propagation chain rather than isolated faults.", failedCount)
		analysis.Confidence = 0.7
		analysis.Recommendations = []Recommendation{
			recommend("Identify the earliest-failed component as the seed of the cascade", "", "Use dependency graph across affected nodes"),
			recommend("Restore the seed component first and verify dependent recovery", "", "Human applies remediations in dependency order"),
		}
	case failedCount >= 1:
		analysis.Archetype = ArchetypeSingleFailure
		analysis.RootCause = "A single component failure is the dominant signal; dependent impact is limited."
		analysis.Confidence = 0.6
		analysis.Recommendations = []Recommendation{
			recommend("Run diagnostics on the failed component before any restart", strings.Join(affected, ","), "Avoid flapping"),
		}
	case len(anomalyMetrics) == 1 && len(anomalyNodes) >= a.clusterThreshold:
		metric := firstMetric(anomalyMetrics)
		analysis.Archetype = ArchetypeClusterPressure
		analysis.RootCause = fmt.Sprintf("%s pressure is elevated across %d nodes concurrently, suggesting a shared upstream constraint rather than per-node faults.", strings.ToUpper(metric), len(anomalyNodes))
		analysis.Confidence = 0.75
		analysis.Recommendations = []Recommendation{
			recommend("Look for a shared dependency the affected nodes have in common", "", "Compare the dependency subgraph"),
			recommend("Committed capacity review for "+strings.ToUpper(metric), "", "Advisory; production resize requires human approval"),
		}
	case degradedCount > 0:
		analysis.Archetype = ArchetypeCapacityBottleneck
		analysis.RootCause = "Nodes are degraded rather than failed; sustained resource pressure indicates a capacity bottleneck."
		analysis.Confidence = 0.65
		analysis.Recommendations = []Recommendation{
			recommend("Check sustained resource ceilings on the degraded nodes", strings.Join(affected, ","), ""),
			recommend("Plan an additive capacity increase", "", "Without approval the advisor will not modify any node"),
		}
	case latencySignals(signals):
		analysis.Archetype = ArchetypeLatencyDegradation
		analysis.RootCause = "Latency-sensitive signals dominate; no component is failed, indicating systemic degradation."
		analysis.Confidence = 0.55
		analysis.Recommendations = []Recommendation{
			recommend("Trace latency across the dependency chain", strings.Join(affected, ","), ""),
		}
	default:
		analysis.RootCause = "No single root cause could be determined from available signals."
		analysis.Confidence = 0.2
		analysis.Recommendations = []Recommendation{
			recommend("Collect more signals before acting", "", "Continuous metric ingestion recommended"),
		}
	}
	return analysis
}

func (a *Advisor) observations(affected []string, snap Snapshot) []string {
	out := make([]string, 0, len(affected))
	seen := make(map[string]bool)
	for _, id := range affected {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if snap == nil {
			out = append(out, fmt.Sprintf("Node %s (unknown state)", id))
			continue
		}
		node, ok := snap.GetNode(id)
		if !ok {
			out = append(out, fmt.Sprintf("Node %s (not registered in topology)", id))
			continue
		}
		out = append(out, fmt.Sprintf("Node %s type=%s state=%s health=%.2f", id, node.Type, node.State, node.Health))
	}
	sort.Strings(out)
	return out
}

func hasChaos(signals []incident.Signal) (bool, []string) {
	var targets []string
	active := false
	for _, s := range signals {
		if s.Kind == "chaos" {
			active = true
			targets = append(targets, s.NodeID)
		}
	}
	return active, targets
}

func metricSpread(signals []incident.Signal) (map[string]int, map[string]bool) {
	metrics := make(map[string]int)
	nodes := make(map[string]bool)
	for _, s := range signals {
		if s.Kind == "anomaly" {
			metrics[s.Metric]++
			nodes[s.NodeID] = true
		}
	}
	return metrics, nodes
}

func stateCounts(ids []string, snap Snapshot) (failed, degraded int) {
	if snap == nil {
		return 0, 0
	}
	for _, id := range ids {
		node, ok := snap.GetNode(id)
		if !ok {
			continue
		}
		switch node.State {
		case models.NodeStateFailed:
			failed++
		case models.NodeStateDegraded:
			degraded++
		}
	}
	return failed, degraded
}

func multipleAffected(ids []string) bool {
	count := 0
	for _, id := range ids {
		if id != "" {
			count++
		}
	}
	return count >= 2
}

func latencySignals(signals []incident.Signal) bool {
	for _, s := range signals {
		if strings.Contains(s.Metric, "latency") || strings.Contains(s.Detail, "latency") {
			return true
		}
	}
	return false
}

func recommend(action, target, note string) Recommendation {
	return Recommendation{Action: action, Target: target, RequiresApproval: true, Note: note}
}

func joinTargets(targets []string) string {
	return strings.Join(targets, ", ")
}

func firstMetric(m map[string]int) string {
	for k := range m {
		return k
	}
	return ""
}
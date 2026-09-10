package models

import (
	"time"
)

type NodeType string

const (
	NodeTypeServer    NodeType = "server"
	NodeTypeVM        NodeType = "vm"
	NodeTypeContainer NodeType = "container"
	NodeTypePod       NodeType = "pod"
	NodeTypeService   NodeType = "service"
	NodeTypeDatabase  NodeType = "database"
	NodeTypeNetwork   NodeType = "network"
	NodeTypeRouter    NodeType = "router"
	NodeTypeSwitch    NodeType = "switch"
	NodeTypeStorage   NodeType = "storage"
	NodeTypeApp       NodeType = "application"
	NodeTypeEndpoint  NodeType = "endpoint"
)

type EdgeType string

const (
	EdgeDependsOn     EdgeType = "depends_on"
	EdgeCommunicates  EdgeType = "communicates_with"
	EdgeHostedOn      EdgeType = "hosted_on"
	EdgeRoutesTo      EdgeType = "routes_to"
	EdgeReplicatesTo  EdgeType = "replicates_to"
	EdgeConnectsTo    EdgeType = "connects_to"
)

type NodeState string

const (
	NodeStateHealthy  NodeState = "healthy"
	NodeStateDegraded NodeState = "degraded"
	NodeStateFailed   NodeState = "failed"
	NodeStateUnknown  NodeState = "unknown"
)

type Node struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         NodeType          `json:"type"`
	State        NodeState         `json:"state"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CPU          float64           `json:"cpu"`
	Memory       float64           `json:"memory"`
	Disk         float64           `json:"disk"`
	Network      float64           `json:"network"`
	Load         float64           `json:"load"`
	WorkloadCount int              `json:"workload_count"`
	Location     string            `json:"location,omitempty"`
	Health       float64           `json:"health"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type Edge struct {
	ID       string    `json:"id"`
	Source   string    `json:"source"`
	Target   string    `json:"target"`
	Type     EdgeType  `json:"type"`
	Weight   float64   `json:"weight"`
	Metadata map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Endpoint   string            `json:"endpoint,omitempty"`
	State      NodeState         `json:"state"`
	NodeID     string            `json:"node_id"`
	Ports      []int             `json:"ports,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type Incident struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Severity        string    `json:"severity"`
	State           string    `json:"state"`
	AffectedNodes   []string  `json:"affected_nodes"`
	AffectedServices []string `json:"affected_services"`
	RootCause       string    `json:"root_cause,omitempty"`
	Resolution      string    `json:"resolution,omitempty"`
	AIAnalysis      string    `json:"ai_analysis,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
}

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusAssigned   JobStatus = "assigned"
	JobStatusRunning    JobStatus = "running"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusDead       JobStatus = "dead"
)

type JobType string

const (
	JobScanNetwork      JobType = "SCAN_NETWORK"
	JobCheckService     JobType = "CHECK_SERVICE"
	JobRunBackup        JobType = "RUN_BACKUP"
	JobAnalyzeLogs      JobType = "ANALYZE_LOGS"
	JobRestartContainer JobType = "RESTART_CONTAINER"
	JobCollectMetrics   JobType = "COLLECT_METRICS"
	JobRunAIAnalysis    JobType = "RUN_AI_ANALYSIS"
	JobGenerateReport   JobType = "GENERATE_REPORT"
)

type Job struct {
	ID          string            `json:"id"`
	Type        JobType           `json:"type"`
	Status      JobStatus         `json:"status"`
	Priority    int               `json:"priority"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	WorkerID    string            `json:"worker_id,omitempty"`
	RetryCount  int               `json:"retry_count"`
	MaxRetries  int               `json:"max_retries"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
}

type Worker struct {
	ID          string    `json:"id"`
	Addr        string    `json:"addr"`
	State       string    `json:"state"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	JobCount    int       `json:"job_count"`
	Capacity    int       `json:"capacity"`
}

type BenchmarkResult struct {
	ID          string                 `json:"id"`
	Algorithm   string                 `json:"algorithm"`
	InputSize   int                    `json:"input_size"`
	Runtime     time.Duration          `json:"runtime"`
	MemoryBytes int64                  `json:"memory_bytes"`
	Throughput  float64                `json:"throughput,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RunAt       time.Time              `json:"run_at"`
}

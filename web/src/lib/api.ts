export const API_BASE = process.env.NEXT_PUBLIC_ATLAS_API_URL ?? "http://localhost:8080/api/v1";

export interface NodeView {
  id: string;
  name: string;
  type: string;
  state: string;
  health: number;
  cpu: number;
  memory: number;
  location?: string;
}

export interface TopologyView {
  nodes: NodeView[];
  edges: { source: string; target: string; type?: string }[];
  health_score: number;
  bottlenecks: string[];
  cycles: boolean;
}

export interface IncidentView {
  id: string;
  title: string;
  description: string;
  severity: string;
  state: string;
  affected_nodes: string[];
  ai_analysis?: string;
  root_cause?: string;
  created_at: string;
}

export interface AnomalyView {
  id: string;
  node_id: string;
  metric: string;
  value: number;
  severity: string;
  z_score: number;
  timestamp: string;
}

export interface ChaosExperimentView {
  id: string;
  type: string;
  target: string;
  status: string;
  duration: number;
  started_at?: string;
  error?: string;
}

export interface MetricsView {
  counters: Record<string, number>;
  gauges: Record<string, number>;
}

export interface JobView {
  id: string;
  name: string;
  type: string;
  status: string;
  priority: number;
  assigned_node?: string;
  created_at: string;
  last_error?: string;
}

export interface ClusterView {
  status: string;
  nodes: number;
  healthy: number;
  degraded: number;
  failed: number;
  health_score: number;
  leader?: string;
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`${path} -> ${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export function getHealth(): Promise<{ status: string; time?: string }> {
  return get("/health");
}

export function getTopology(): Promise<TopologyView> {
  return get("/topology");
}

export function getNodes(): Promise<NodeView[]> {
  return get("/nodes");
}

export function getIncidents(open = true): Promise<{ incidents: IncidentView[]; total: number }> {
  return get(`/incidents${open ? "?open=true" : ""}`);
}

export function getAnomalies(): Promise<{ anomalies: AnomalyView[]; total: number }> {
  return get("/anomalies");
}

export function getChaos(): Promise<{ experiments: ChaosExperimentView[]; total: number }> {
  return get("/chaos");
}

export function getChaosActive(): Promise<{ experiments: ChaosExperimentView[]; total: number }> {
  return get("/chaos/active");
}

export function getMetrics(): Promise<MetricsView> {
  return get("/metrics");
}

export function getJobs(): Promise<{ jobs: JobView[]; total: number; stats?: Record<string, unknown> }> {
  return get("/jobs");
}

export function getCluster(): Promise<ClusterView> {
  return get("/cluster");
}
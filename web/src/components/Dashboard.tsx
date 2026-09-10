"use client";

import { useEffect, useState } from "react";
import * as api from "@/lib/api";

type LoadState<T> = { data: T | null; error: string | null };

function useData<T>(loader: () => Promise<T>, refreshMs = 5000): LoadState<T> {
  const [state, setState] = useState<LoadState<T>>({ data: null, error: null });

  useEffect(() => {
    let active = true;
    const run = async () => {
      try {
        const data = await loader();
        if (active) setState({ data, error: null });
      } catch (err) {
        if (active) setState({ data: null, error: String(err) });
      }
    };
    run();
    const id = setInterval(run, refreshMs);
    return () => {
      active = false;
      clearInterval(id);
    };
  }, [loader, refreshMs]);

  return state;
}

function Card({
  title,
  children,
  className = "",
}: {
  title: string;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <section
      className={`rounded-xl border border-zinc-800 bg-zinc-900/50 p-4 ${className}`}
    >
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wider text-zinc-400">
        {title}
      </h2>
      {children}
    </section>
  );
}

function healthColor(health: number): string {
  if (health <= 0.2) return "text-red-400";
  if (health <= 0.6) return "text-amber-400";
  return "text-emerald-400";
}

function stateColor(state: string): string {
  switch (state) {
    case "healthy":
      return "bg-emerald-500/15 text-emerald-400";
    case "degraded":
      return "bg-amber-500/15 text-amber-400";
    case "failed":
      return "bg-red-500/15 text-red-400";
    default:
      return "bg-zinc-500/15 text-zinc-400";
  }
}

function severityColor(severity: string): string {
  return severity === "critical" ? "text-red-400" : "text-amber-400";
}

export default function Dashboard() {
  const health = useData(api.getHealth);
  const cluster = useData(api.getCluster);
  const topo = useData(api.getTopology);
  const nodes = useData(api.getNodes);
  const incidents = useData(() => api.getIncidents(true));
  const anomalies = useData(api.getAnomalies);
  const chaos = useData(api.getChaosActive);
  const metrics = useData(api.getMetrics);

  const topoNodes = topo.data?.nodes ?? nodes.data ?? [];

  return (
    <div className="mx-auto max-w-7xl space-y-6 px-6 py-8">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white">
            <span className="text-emerald-400">Atlas</span> Dashboard
          </h1>
          <p className="text-sm text-zinc-500">
            Distributed Infrastructure Intelligence
          </p>
        </div>
        <div className="flex items-center gap-6 text-sm">
          <StatusChip label="API" ok={!!health.data && health.data.status === "healthy"} />
          <div className="text-zinc-400">
            Health score{" "}
            <span className={`text-lg font-semibold ${healthColor(cluster.data?.health_score ?? 1)}`}>
              {(cluster.data?.health_score ?? 1).toFixed(2)}
            </span>
          </div>
          <div className="text-zinc-400">
            Goroutines{" "}
            <span className="font-semibold text-zinc-200">
              {Math.round(metrics.data?.gauges.atlas_runtime_goroutines ?? 0)}
            </span>
          </div>
        </div>
      </header>

      {health.error && (
        <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300">
          Cannot reach the Atlas API at {api.API_BASE} — start the server with{" "}
          <code className="text-red-200">go run ./cmd/atlas-server</code>. ({health.error})
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2 xl:grid-cols-3">
        <Card title="Components">
          <Kpis
            items={[
              { label: "Nodes", value: cluster.data?.nodes ?? topoNodes.length },
              { label: "Healthy", value: cluster.data?.healthy },
              { label: "Degraded", value: cluster.data?.degraded },
              { label: "Failed", value: cluster.data?.failed },
            ]}
          />
        </Card>

        <Card title="Topology">
          <Kpis
            items={[
              { label: "Edges", value: topo.data?.edges.length },
              { label: "Bottlenecks", value: topo.data?.bottlenecks.length },
            ]}
          />
          <p className="mt-2 text-xs text-zinc-500">
            Cycles detected:{" "}
            <span className={topo.data?.cycles ? "text-amber-400" : "text-emerald-400"}>
              {topo.data?.cycles ? "yes" : "no"}
            </span>
          </p>
        </Card>

        <Card title="Open Incidents">
          {incidents.data?.incidents.length ? (
            <ul className="space-y-3">
              {incidents.data.incidents.slice(0, 5).map((inc) => (
                <li key={inc.id} className="rounded-lg border border-zinc-800 bg-zinc-950 p-3">
                  <div className="flex items-center justify-between">
                    <span className={`text-sm font-medium ${severityColor(inc.severity)}`}>
                      {inc.title}
                    </span>
                    <span className="text-xs text-zinc-500">{inc.id}</span>
                  </div>
                  {inc.ai_analysis ? (
                    <p className="mt-1 text-xs leading-relaxed text-zinc-400 line-clamp-2">
                      {inc.ai_analysis}
                    </p>
                  ) : (
                    <p className="mt-1 text-xs text-zinc-600">Not analyzed yet</p>
                  )}
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-zinc-600">No open incidents</p>
          )}
        </Card>

        <Card title="Anomalies" className="xl:col-span-1">
          {anomalies.data?.anomalies.length ? (
            <ul className="space-y-2">
              {anomalies.data.anomalies.slice(0, 5).map((a) => (
                <li
                  key={a.id}
                  className="flex items-center justify-between rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm"
                >
                  <div>
                    <span className={`font-medium ${severityColor(a.severity)}`}>
                      {a.node_id}
                    </span>
                    <span className="text-zinc-500"> / {a.metric}</span>
                  </div>
                  <span className="text-xs text-zinc-400">
                    {a.value.toFixed(1)} (z={a.z_score.toFixed(2)})
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-zinc-600">No anomalies recorded</p>
          )}
        </Card>

        <Card title="Running Chaos Experiments">
          {chaos.data?.experiments.length ? (
            <ul className="space-y-2">
              {chaos.data.experiments.map((e) => (
                <li
                  key={e.id}
                  className="flex justify-between rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm"
                >
                  <span className="text-amber-300">{e.type}</span>
                  <span className="text-zinc-400">→ {e.target}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-zinc-600">No experiments running</p>
          )}
        </Card>

        <Card title="Jobs" className="xl:col-span-1">
          <JobsPanel />
        </Card>
      </div>

      <Card title="Nodes" className="lg:col-span-2 xl:col-span-3">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-zinc-500">
                <th className="pb-2 pr-4">ID</th>
                <th className="pb-2 pr-4">Type</th>
                <th className="pb-2 pr-4">State</th>
                <th className="pb-2 pr-4 text-right">Health</th>
                <th className="pb-2 pr-4 text-right">CPU</th>
                <th className="pb-2 pr-4 text-right">Memory</th>
              </tr>
            </thead>
            <tbody>
              {topoNodes.map((n) => (
                <tr key={n.id} className="border-t border-zinc-800/60 text-zinc-300">
                  <td className="py-2 pr-4 font-medium text-white">{n.id}</td>
                  <td className="py-2 pr-4 text-zinc-500">{n.type}</td>
                  <td className="py-2 pr-4">
                    <span className={`rounded-full px-2 py-0.5 text-xs ${stateColor(n.state)}`}>
                      {n.state}
                    </span>
                  </td>
                  <td className={`py-2 pr-4 text-right font-semibold ${healthColor(n.health)}`}>
                    {n.health.toFixed(2)}
                  </td>
                  <td className="py-2 pr-4 text-right">{n.cpu.toFixed(1)}</td>
                  <td className="py-2 pr-4 text-right">{n.memory.toFixed(1)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}

function StatusChip({ label, ok }: { label: string; ok: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <span
        className={`h-2.5 w-2.5 rounded-full ${ok ? "bg-emerald-400" : "bg-red-500"}`}
      />
      <span className="text-zinc-400">{label}</span>
    </div>
  );
}

function Kpis({ items }: { items: { label: string; value?: number }[] }) {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      {items.map((it) => (
        <div key={it.label} className="rounded-lg bg-zinc-950 px-3 py-3 text-center">
          <div className="text-2xl font-semibold text-white">{it.value ?? "–"}</div>
          <div className="mt-1 text-xs uppercase tracking-wider text-zinc-500">
            {it.label}
          </div>
        </div>
      ))}
    </div>
  );
}

function JobsPanel() {
  const jobs = useData(api.getJobs);
  if (jobs.error) return <p className="text-sm text-zinc-600">Cannot reach API</p>;
  return (
    <ul className="space-y-2">
      {(jobs.data?.jobs ?? []).slice(0, 5).map((j) => (
        <li
          key={j.id}
          className="flex justify-between rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm"
        >
          <span className="text-zinc-200">{j.name || j.type}</span>
          <span className="text-zinc-500">
            {j.status}
            {j.assigned_node ? ` · ${j.assigned_node}` : ""}
          </span>
        </li>
      ))}
      {!jobs.data?.jobs.length && <p className="text-sm text-zinc-600">No jobs in queue</p>}
    </ul>
  );
}
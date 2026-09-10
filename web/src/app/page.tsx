export default function Home() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-zinc-950 text-white font-sans">
      <main className="flex flex-col items-center gap-8 text-center px-6">
        <div className="text-6xl font-bold tracking-tight">
          <span className="text-emerald-400">Atlas</span>
        </div>
        <p className="max-w-xl text-lg text-zinc-400 leading-relaxed">
          Distributed Infrastructure Intelligence Engine
        </p>
        <div className="flex flex-wrap justify-center gap-3 text-sm text-zinc-500">
          {[
            "Graph Engine",
            "Raft Consensus",
            "Consistent Hashing",
            "Job Queue",
            "Chaos Engine",
            "Anomaly Detection",
            "AI Analysis",
            "Algorithm Playground",
          ].map((f) => (
            <span
              key={f}
              className="rounded-full border border-zinc-800 px-3 py-1 text-xs"
            >
              {f}
            </span>
          ))}
        </div>
        <div className="mt-8 flex gap-4">
          <a
            href="/dashboard"
            className="rounded-lg bg-emerald-600 px-6 py-3 text-sm font-medium text-white transition-colors hover:bg-emerald-500"
          >
            Open Dashboard
          </a>
          <a
            href="https://github.com/atlas-engine/atlas"
            className="rounded-lg border border-zinc-700 px-6 py-3 text-sm font-medium text-zinc-300 transition-colors hover:border-zinc-500"
          >
            View Source
          </a>
        </div>
      </main>
    </div>
  );
}

import type { Stats } from "@/lib/types";
import { Loader2, Clock, Server, CheckCircle, XCircle, FileText, Activity } from "lucide-react";
import type { ReactNode } from "react";

interface Props {
  status: string;
  stats?: Stats;
  query: string;
  summary?: string;
}

function statusTone(status: string): string {
  if (status === "done") return "border-emerald-400/20 bg-emerald-500/10 text-emerald-100";
  if (status === "failed" || status === "cancelled") return "border-rose-400/20 bg-rose-500/10 text-rose-100";
  return "border-cyan-400/20 bg-cyan-500/10 text-cyan-100";
}

export default function SearchStatus({ status, stats, query, summary }: Props) {
  const isRunning = status === "running" || status === "pending";
  const label = isRunning ? "Scanning live sources" : status === "done" ? "Search complete" : "Search failed";

  return (
    <section className="overflow-hidden rounded-[2rem] border border-white/10 bg-surface-900/75 shadow-[0_24px_80px_rgba(2,6,23,0.45)] backdrop-blur-xl">
      <div className="border-b border-white/5 px-5 py-4 sm:px-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
          <div className="min-w-0">
            <div className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold ${statusTone(status)}`}>
              {isRunning ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : status === "done" ? <CheckCircle className="h-3.5 w-3.5" /> : <XCircle className="h-3.5 w-3.5" />}
              {label}
            </div>
            <p className="mt-3 text-xl font-semibold text-surface-50">{query}</p>
            <p className="mt-1 max-w-3xl text-sm leading-6 text-surface-400">
              {isRunning
                ? "Collectors are running and results are streaming into the workspace in real time."
                : status === "done"
                  ? "The investigation finished and the current result set is ready for review."
                  : "The investigation did not complete successfully. Review the search context and try again."}
            </p>
          </div>

          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:min-w-[22rem]">
            <StatCard label="Collectors" value={stats ? `${stats.collectors_ok}/${stats.collectors_run}` : "-"} icon={<Server className="h-4 w-4" />} />
            <StatCard label="Results" value={stats ? `${stats.results_found}` : "-"} icon={<FileText className="h-4 w-4" />} />
            <StatCard label="Duration" value={stats?.duration_ms ? `${stats.duration_ms}ms` : "-"} icon={<Clock className="h-4 w-4" />} />
          </div>
        </div>
      </div>

      {isRunning && (
        <div className="px-5 pt-5 sm:px-6">
          <div className="h-2 overflow-hidden rounded-full bg-surface-800">
            <div className="h-full w-2/3 rounded-full bg-gradient-to-r from-cyan-400 via-blue-400 to-amber-400 shadow-[0_0_28px_rgba(34,211,238,0.35)]" />
          </div>
        </div>
      )}

      <div className="space-y-4 px-5 py-5 sm:px-6">
        {stats && stats.collectors_failed > 0 && (
          <div className="inline-flex items-center gap-2 rounded-full border border-rose-400/20 bg-rose-500/10 px-3 py-1 text-xs font-medium text-rose-100">
            <XCircle className="h-3.5 w-3.5" />
            {stats.collectors_failed} collectors failed
          </div>
        )}

        {summary && (
          <div className="overflow-hidden rounded-[1.75rem] border border-cyan-400/15 bg-gradient-to-br from-cyan-500/10 via-surface-950/70 to-amber-500/10 p-4 shadow-[0_18px_50px_rgba(2,6,23,0.35)]">
            <div className="flex items-start gap-4">
              <div className="mt-1 flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-cyan-400/20 bg-cyan-400/10 text-cyan-200">
                <Activity className="h-4.5 w-4.5" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="text-xs font-semibold uppercase tracking-[0.22em] text-cyan-100">AI summary</p>
                  <span className="rounded-full border border-white/10 bg-white/5 px-2.5 py-0.5 text-[10px] uppercase tracking-[0.18em] text-surface-400">
                    condensed view
                  </span>
                </div>
                <p className="mt-2 text-sm leading-7 text-surface-200 whitespace-pre-wrap">
                  {summary}
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </section>
  );
}

function StatCard({ label, value, icon }: { label: string; value: string; icon: ReactNode }) {
  return (
    <div className="rounded-2xl border border-white/5 bg-surface-950/50 p-3 shadow-inner shadow-black/20">
      <div className="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-surface-500">
        {icon}
        {label}
      </div>
      <div className="mt-2 text-lg font-semibold text-surface-50">{value}</div>
    </div>
  );
}

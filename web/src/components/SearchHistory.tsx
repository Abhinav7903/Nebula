"use client";

import { Clock, Trash2, ChevronRight, History } from "lucide-react";
import type { SearchSummary } from "@/lib/types";

interface Props {
  searches: SearchSummary[];
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
  activeId?: string;
  className?: string;
}

const TYPE_ICONS: Record<string, string> = {
  ipv4: "🌐",
  ipv6: "🌐",
  domain: "🌍",
  subdomain: "🔗",
  email: "📧",
  url: "🔗",
  username: "👤",
  phone: "📞",
  bitcoin_address: "₿",
  ethereum_address: "♦",
  solana_address: "◎",
  hash_md5: "#",
  hash_sha1: "#",
  hash_sha256: "#",
};

export default function SearchHistory({ searches, onSelect, onDelete, activeId, className }: Props) {
  if (searches.length === 0) return null;

  const sorted = [...searches].sort(
    (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
  );

  return (
    <aside
      className={`rounded-[2rem] border border-white/10 bg-surface-900/75 p-4 shadow-[0_24px_80px_rgba(2,6,23,0.35)] backdrop-blur-xl ${className || ""}`}
    >
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <p className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface-950/60 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.22em] text-surface-300">
            <History className="h-3.5 w-3.5 text-cyan-300" />
            Recent searches
          </p>
          <h3 className="mt-3 text-lg font-semibold text-surface-50">Investigation history</h3>
          <p className="mt-1 text-sm text-surface-400">Return to a previous query or remove stale investigations.</p>
        </div>
        <span className="rounded-full border border-surface-700/80 bg-surface-950/60 px-3 py-1 text-xs text-surface-400">
          {sorted.length}
        </span>
      </div>

      <div className="space-y-2">
        {sorted.map((s, index) => {
          const active = s.search_id === activeId;

          return (
            <div
              key={s.search_id}
              role="button"
              tabIndex={0}
              onClick={() => onSelect(s.search_id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onSelect(s.search_id);
                }
              }}
              className={`history-enter group flex items-center gap-3 rounded-2xl border px-3 py-3 text-sm transition-all duration-200 ${
                active
                  ? "border-cyan-400/25 bg-cyan-500/10 text-surface-50 shadow-[0_0_0_1px_rgba(34,211,238,0.14)]"
                  : "border-white/5 bg-surface-950/40 text-surface-300 hover:-translate-y-0.5 hover:border-white/10 hover:bg-surface-800/80"
              }`}
              style={{ animationDelay: `${index * 40}ms` }}
            >
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-white/5 bg-surface-950/70 text-base">
                {TYPE_ICONS[s.query_type] || "?"}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="truncate font-mono text-xs">{s.query}</p>
                  <span className={`rounded-full px-2 py-0.5 text-[10px] uppercase tracking-[0.18em] ${active ? "bg-cyan-500/15 text-cyan-200" : "bg-surface-800 text-surface-500"}`}>
                    {s.status}
                  </span>
                </div>
                <p className="mt-1 flex items-center gap-1.5 text-xs text-surface-500">
                  <Clock className="h-3.5 w-3.5" />
                  {new Date(s.started_at).toLocaleString()}
                </p>
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(s.search_id);
                }}
                className="inline-flex h-8 w-8 items-center justify-center rounded-xl text-surface-500 opacity-0 transition-all hover:bg-rose-500/10 hover:text-rose-300 group-hover:opacity-100"
                aria-label={`Delete search ${s.query}`}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
              <ChevronRight className={`h-4 w-4 shrink-0 transition-transform ${active ? "text-cyan-300" : "text-surface-500 group-hover:translate-x-0.5"}`} />
            </div>
          );
        })}
      </div>
    </aside>
  );
}

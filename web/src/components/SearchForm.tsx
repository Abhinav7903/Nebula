"use client";

import { useState, type FormEvent } from "react";
import { Search, Loader2, Sparkles, ArrowRight } from "lucide-react";

interface Props {
  onSearch: (query: string) => void;
  loading: boolean;
}

const PLACEHOLDER = "Search domain, IP, email, URL, hash, username, crypto address...";

const QUICK_HINTS = ["Domain", "Email", "URL", "IP"];

export default function SearchForm({ onSearch, loading }: Props) {
  const [query, setQuery] = useState("");

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    const q = query.trim();
    if (!q || loading) return;
    onSearch(q);
  };

  return (
    <form onSubmit={handleSubmit} className={`w-full ${loading ? "search-activity" : ""}`}>
      <div className="rounded-[2rem] border border-white/10 bg-surface-900/75 p-4 shadow-[0_24px_80px_rgba(2,6,23,0.55)] backdrop-blur-xl sm:p-5">
        <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="inline-flex items-center gap-2 rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.24em] text-cyan-200">
              <Sparkles className="h-3.5 w-3.5" />
              Investigation search
            </p>
            <h2 className="mt-3 text-xl font-semibold text-surface-50 sm:text-2xl">
              Start a live intelligence sweep
            </h2>
            <p className="mt-1 max-w-2xl text-sm leading-6 text-surface-400">
              Query public intelligence sources and stream results into a focused workspace.
            </p>
          </div>
          <div className="hidden flex-wrap items-center gap-2 sm:flex">
            {QUICK_HINTS.map((hint) => (
              <span
                key={hint}
                className="rounded-full border border-surface-700/80 bg-surface-800/70 px-3 py-1 text-xs text-surface-300"
              >
                {hint}
              </span>
            ))}
          </div>
        </div>

        <div className="relative flex flex-col gap-3 sm:block">
          <Search className="pointer-events-none absolute left-4 top-4 h-5 w-5 text-surface-500" />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={PLACEHOLDER}
            disabled={loading}
            aria-busy={loading}
            className="w-full rounded-2xl border border-surface-700/90 bg-surface-950/80 py-4 pl-12 pr-4 text-[15px] text-surface-50 placeholder-surface-500 shadow-inner shadow-black/20 transition-all focus:border-cyan-400 focus:outline-none focus:ring-2 focus:ring-cyan-400/20 disabled:cursor-not-allowed disabled:opacity-60 sm:pr-32"
          />
          <button
            type="submit"
            disabled={loading || !query.trim()}
            className={`inline-flex w-full items-center justify-center gap-2 rounded-2xl bg-gradient-to-r from-cyan-500 to-blue-500 px-4 py-3 text-sm font-semibold text-slate-950 shadow-lg shadow-cyan-950/30 transition-all hover:brightness-110 disabled:cursor-not-allowed disabled:from-surface-700 disabled:to-surface-700 disabled:text-surface-400 sm:absolute sm:right-2 sm:top-1/2 sm:w-auto sm:-translate-y-1/2 sm:py-2.5 ${loading ? "search-button-active" : ""}`}
          >
            {loading ? <Loader2 className="h-4.5 w-4.5 animate-spin" /> : <ArrowRight className="h-4.5 w-4.5" />}
            <span>Analyze</span>
          </button>
        </div>

        <p className="mt-3 text-xs leading-5 text-surface-500">
          Supports domains, IPs, emails, URLs, hashes, crypto addresses, usernames, and more.
        </p>
      </div>
    </form>
  );
}

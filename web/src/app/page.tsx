"use client";

import { useEffect, useCallback, useRef, useState } from "react";
import { Search as SearchIcon, Network, Sparkles, Radar, Shield, Activity, ArrowUpRight, Menu, X } from "lucide-react";
import SearchForm from "@/components/SearchForm";
import SearchStatus from "@/components/SearchStatus";
import SearchStatusSkeleton from "@/components/SearchStatusSkeleton";
import SearchHistory from "@/components/SearchHistory";
import ResultGrid from "@/components/ResultGrid";
import ResultGridSkeleton from "@/components/ResultGridSkeleton";
import { createSearch, getSearch, listSearches, deleteSearch, streamSearch } from "@/lib/api";
import type { Result } from "@/lib/types";
import {
  beginSearch,
  clearActiveSearch,
  clearLiveResults,
  mergeLiveResults,
  removeRecentSearch,
  replaceLiveResults,
  setActiveId,
  setActiveSearch,
  setLoading,
  setRecentSearches,
} from "@/store/searchSlice";
import { useAppDispatch, useAppSelector } from "@/store/hooks";

const featureBadges = ["Live streaming", "Multi-source", "Fast lookup", "Actionable context"];

export default function Home() {
  const dispatch = useAppDispatch();
  const { loading, activeId, activeSearch, liveResults, recentSearches } = useAppSelector(
    (state) => state.search
  );
  const [isHistoryOpen, setIsHistoryOpen] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const closeStreamRef = useRef<(() => void) | null>(null);

  const loadRecent = useCallback(async () => {
    try {
      const list = await listSearches();
      dispatch(setRecentSearches(list));
    } catch {
      // ignore
    }
  }, [dispatch]);

  useEffect(() => {
    loadRecent();
  }, [loadRecent]);

  useEffect(() => {
    if (typeof document === "undefined") return;
    document.body.style.overflow = isHistoryOpen ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [isHistoryOpen]);

  const cleanup = useCallback(() => {
    if (closeStreamRef.current) {
      closeStreamRef.current();
      closeStreamRef.current = null;
    }
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  }, []);

  const startPolling = useCallback(
    (id: string) => {
      cleanup();
      const interval = setInterval(async () => {
        try {
          const s = await getSearch(id);
          dispatch(setActiveSearch(s));
          dispatch(mergeLiveResults(s.results));
          if (s.status !== "running" && s.status !== "pending") {
            if (interval) clearInterval(interval);
            pollRef.current = null;
            dispatch(setLoading(false));
            loadRecent();
          }
        } catch {
          // ignore
        }
      }, 1000);
      pollRef.current = interval;
    },
    [cleanup, dispatch, loadRecent]
  );

  const handleSearch = useCallback(
    async (query: string) => {
      dispatch(beginSearch());
      cleanup();

      try {
        const { search_id } = await createSearch(query);
        dispatch(setActiveId(search_id));
        setIsHistoryOpen(false);

        closeStreamRef.current = streamSearch(
          search_id,
          (event, payload) => {
            if (event === "collector_result" && payload?.result) {
              const r = payload.result as Result;
              dispatch(mergeLiveResults([r]));
            }
          },
          () => {
            startPolling(search_id);
          }
        );

        startPolling(search_id);
      } catch {
        dispatch(setLoading(false));
      }
    },
    [cleanup, dispatch, startPolling]
  );

  const handleSelect = useCallback(
    async (id: string) => {
      dispatch(setActiveId(id));
      dispatch(setLoading(false));
      dispatch(setActiveSearch(null));
      dispatch(clearLiveResults());
      cleanup();
      setIsHistoryOpen(false);

      try {
        const s = await getSearch(id);
        dispatch(setActiveSearch(s));
        dispatch(replaceLiveResults(s.results));
        if (s.status === "running" || s.status === "pending") {
          startPolling(id);
        }
      } catch {
        // ignore
      }
    },
    [cleanup, dispatch, startPolling]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await deleteSearch(id);
        dispatch(removeRecentSearch(id));
        if (activeId === id) {
          dispatch(clearActiveSearch());
          cleanup();
        }
        loadRecent();
      } catch {
        // ignore
      }
    },
    [activeId, cleanup, dispatch, loadRecent]
  );

  const stats = activeSearch?.stats;
  const showStatusSkeleton = loading && !activeSearch;
  const showResultSkeleton = loading && liveResults.length === 0;

  return (
    <div className="relative min-h-screen overflow-hidden">
      <div className="pointer-events-none absolute inset-x-0 top-0 h-[34rem] bg-[radial-gradient(circle_at_20%_15%,rgba(20,184,166,0.16),transparent_35%),radial-gradient(circle_at_80%_10%,rgba(249,115,22,0.12),transparent_28%),radial-gradient(circle_at_50%_40%,rgba(56,189,248,0.06),transparent_45%)]" />
      <div className="pointer-events-none absolute inset-x-0 top-[22rem] h-px bg-gradient-to-r from-transparent via-white/10 to-transparent" />

      <header className="sticky top-0 z-30 border-b border-white/8 bg-surface-950/55 backdrop-blur-xl">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl border border-cyan-400/20 bg-cyan-400/10 shadow-lg shadow-cyan-950/20">
              <Network className="h-6 w-6 text-cyan-300" />
            </div>
            <div>
              <h1 className="text-base font-semibold tracking-wide text-surface-50">Nebula</h1>
              <p className="text-xs text-surface-400">OSINT intelligence workspace</p>
            </div>
          </div>

          <div className="hidden items-center gap-2 md:flex">
            <span className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface-900/60 px-3 py-1.5 text-xs text-surface-300">
              <Shield className="h-3.5 w-3.5 text-emerald-300" />
              Secure collection
            </span>
            <span className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface-900/60 px-3 py-1.5 text-xs text-surface-300">
              <Activity className="h-3.5 w-3.5 text-cyan-300" />
              {recentSearches.length} recent searches
            </span>
          </div>

          <button
            type="button"
            onClick={() => setIsHistoryOpen(true)}
            className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface-900/70 px-3 py-2 text-xs font-medium text-surface-200 shadow-lg shadow-black/10 transition-colors hover:border-cyan-400/20 hover:bg-cyan-400/10 hover:text-cyan-100 lg:hidden"
            aria-label="Open search history"
          >
            <Menu className="h-4 w-4" />
            History
          </button>
        </div>
      </header>

      <main className="relative mx-auto max-w-7xl px-4 pb-10 pt-8 sm:px-6 lg:px-8 lg:pt-10">
        <section className="grid gap-6 lg:grid-cols-[minmax(0,1.6fr)_minmax(18rem,0.8fr)]">
          <div className="space-y-6">
            <div className="panel-enter rounded-[2rem] border border-white/10 bg-surface-900/55 p-6 shadow-[0_24px_100px_rgba(2,6,23,0.45)] backdrop-blur-xl sm:p-7">
              <div className="flex flex-wrap items-center gap-2">
                <span className="inline-flex items-center gap-2 rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.24em] text-cyan-200">
                  <Sparkles className="h-3.5 w-3.5" />
                  Intelligence console
                </span>
                <span className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface-950/60 px-3 py-1 text-[11px] font-medium uppercase tracking-[0.2em] text-surface-400">
                  <Radar className="h-3.5 w-3.5 text-amber-300" />
                  Unified search flow
                </span>
              </div>

              <div className="mt-6 max-w-3xl">
                <h2 className="text-3xl font-semibold tracking-tight text-surface-50 sm:text-5xl sm:leading-tight">
                  Track entities, sources, and evidence in one focused workspace.
                </h2>
                <p className="mt-4 max-w-2xl text-sm leading-7 text-surface-400 sm:text-base">
                  Nebula streams results from multiple collectors, keeps recent investigations visible, and turns raw data into a cleaner review experience.
                </p>
              </div>

              <div className="mt-6 flex flex-wrap gap-2">
                {featureBadges.map((badge) => (
                  <span
                    key={badge}
                    className="rounded-full border border-white/8 bg-white/5 px-3 py-1.5 text-xs text-surface-300"
                  >
                    {badge}
                  </span>
                ))}
              </div>
            </div>

            <SearchForm onSearch={handleSearch} loading={loading} />

            <div className="grid gap-3 sm:grid-cols-3">
              <StatStrip label="Live results" value={String(liveResults.length)} hint="Streaming in the current session" />
              <StatStrip label="Recent searches" value={String(recentSearches.length)} hint="Quick resume on the right" />
              <StatStrip
                label="Search state"
                value={loading ? "Running" : activeSearch ? activeSearch.status : "Idle"}
                hint={loading ? "Collectors are working" : "Ready for a new query"}
              />
            </div>
          </div>

          <aside className="space-y-6">
            <div className="panel-enter rounded-[2rem] border border-white/10 bg-surface-900/55 p-5 shadow-[0_24px_100px_rgba(2,6,23,0.4)] backdrop-blur-xl">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-[0.24em] text-surface-500">Overview</p>
                  <h3 className="mt-2 text-lg font-semibold text-surface-50">Active investigation</h3>
                </div>
                <ArrowUpRight className="h-5 w-5 text-surface-500" />
              </div>

              <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
                <MiniMetric label="Query mode" value={activeSearch ? activeSearch.query_type : "Awaiting input"} />
                <MiniMetric label="Results visible" value={String(liveResults.length)} />
                <MiniMetric label="Collectors run" value={stats ? `${stats.collectors_ok}/${stats.collectors_run}` : "-"} />
                <MiniMetric label="Duration" value={stats?.duration_ms ? `${stats.duration_ms}ms` : "-"} />
              </div>
            </div>

            <div className="hidden lg:block">
              <SearchHistory
                searches={recentSearches}
                onSelect={handleSelect}
                onDelete={handleDelete}
                activeId={activeId || undefined}
              />
            </div>
          </aside>
        </section>

        <section className="mt-8 grid gap-6 lg:grid-cols-[minmax(0,1fr)_18rem]">
          <div className="min-w-0 space-y-6">
            {activeId && activeSearch && (
              <div className="panel-enter">
                <SearchStatus
                  status={activeSearch.status}
                  stats={activeSearch.stats}
                  query={activeSearch.query}
                  summary={activeSearch.summary}
                />
              </div>
            )}

            {showStatusSkeleton && (
              <div className="panel-enter">
                <SearchStatusSkeleton />
              </div>
            )}

            {liveResults.length > 0 ? (
              <div className="panel-enter space-y-4">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <h2 className="text-sm font-semibold uppercase tracking-[0.24em] text-surface-500">
                      Result stream
                    </h2>
                    <p className="mt-1 text-sm text-surface-400">{liveResults.length} findings are currently visible.</p>
                  </div>
                </div>
                <ResultGrid results={liveResults} />
              </div>
            ) : showResultSkeleton ? (
              <div className="panel-enter space-y-4">
                <div>
                  <h2 className="text-sm font-semibold uppercase tracking-[0.24em] text-surface-500">
                    Result stream
                  </h2>
                  <p className="mt-1 text-sm text-surface-400">Collectors are starting up. This area will fill as matches arrive.</p>
                </div>
                <ResultGridSkeleton />
              </div>
            ) : activeId && activeSearch && activeSearch.status !== "running" && activeSearch.status !== "pending" ? (
              <div className="panel-enter rounded-[2rem] border border-white/10 bg-surface-900/60 p-8 text-surface-400 shadow-[0_24px_80px_rgba(2,6,23,0.35)] backdrop-blur-xl">
                <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-white/10 bg-surface-950/70">
                  <SearchIcon className="h-6 w-6 text-surface-500" />
                </div>
                <p className="mt-4 text-base font-medium text-surface-100">No results found</p>
                <p className="mt-1 max-w-xl text-sm leading-6 text-surface-400">
                  This search completed without returning any collector matches. Try a broader query or a different entity type.
                </p>
              </div>
            ) : !activeId && !loading ? (
              <div className="panel-enter rounded-[2rem] border border-white/10 bg-surface-900/50 p-8 text-surface-400 shadow-[0_24px_80px_rgba(2,6,23,0.3)] backdrop-blur-xl sm:p-10">
                <div className="flex h-14 w-14 items-center justify-center rounded-2xl border border-cyan-400/20 bg-cyan-400/10">
                  <SearchIcon className="h-6 w-6 text-cyan-300" />
                </div>
                <p className="mt-4 text-base font-medium text-surface-100">Enter a query to begin</p>
                <p className="mt-1 max-w-2xl text-sm leading-6 text-surface-400">
                  Search domains, IPs, emails, URLs, hashes, crypto addresses, usernames, and more to build a faster investigation trail.
                </p>
              </div>
            ) : null}
          </div>

          <div className="lg:hidden">
            <div className="rounded-[2rem] border border-white/10 bg-surface-900/55 p-4 shadow-[0_24px_80px_rgba(2,6,23,0.35)] backdrop-blur-xl">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-[0.24em] text-surface-500">History</p>
                  <p className="mt-1 text-sm text-surface-400">Open your recent searches in a slide-up panel.</p>
                </div>
                <button
                  type="button"
                  onClick={() => setIsHistoryOpen(true)}
                  className="inline-flex items-center gap-2 rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-2 text-xs font-medium text-cyan-100"
                >
                  <Menu className="h-4 w-4" />
                  Open
                </button>
              </div>
            </div>
          </div>
        </section>
      </main>

      {isHistoryOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <button
            type="button"
            className="absolute inset-0 bg-surface-950/70 backdrop-blur-sm"
            aria-label="Close search history"
            onClick={() => setIsHistoryOpen(false)}
          />
          <div className="absolute inset-x-0 bottom-0 max-h-[84vh] overflow-hidden rounded-t-[2rem] border-t border-white/10 bg-surface-950/96 shadow-[0_-24px_80px_rgba(2,6,23,0.65)]">
            <div className="flex items-center justify-between border-b border-white/5 px-4 py-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-[0.24em] text-surface-500">Recent searches</p>
                <p className="mt-1 text-sm text-surface-400">Tap a query to reopen it.</p>
              </div>
              <button
                type="button"
                onClick={() => setIsHistoryOpen(false)}
                className="inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/10 bg-surface-900/80 text-surface-300"
                aria-label="Close history panel"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="max-h-[calc(84vh-5rem)] overflow-auto p-4">
              <SearchHistory
                searches={recentSearches}
                onSelect={handleSelect}
                onDelete={handleDelete}
                activeId={activeId || undefined}
                className="border-0 bg-transparent p-0 shadow-none backdrop-blur-0"
              />
              {recentSearches.length === 0 && (
                <div className="rounded-[2rem] border border-white/8 bg-surface-900/50 p-6 text-sm text-surface-400">
                  No recent searches yet.
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function StatStrip({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div className="rounded-2xl border border-white/8 bg-surface-900/60 p-4 shadow-[0_12px_40px_rgba(2,6,23,0.25)] backdrop-blur">
      <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-surface-500">{label}</p>
      <p className="mt-2 text-lg font-semibold text-surface-50">{value}</p>
      <p className="mt-1 text-xs leading-5 text-surface-400">{hint}</p>
    </div>
  );
}

function MiniMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/8 bg-surface-950/55 p-4">
      <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-surface-500">{label}</p>
      <p className="mt-2 break-words text-sm font-semibold text-surface-50">{value}</p>
    </div>
  );
}

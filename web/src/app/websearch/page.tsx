"use client";

import { useState, useCallback } from "react";
import { Search, Globe, Link, ExternalLink, Loader2, ChevronDown, ChevronUp, FileText, AlertCircle, Brain } from "lucide-react";
import { webSearch } from "@/lib/api";
import type { WebSearchResult, FetchedPage, WebSearchResponse } from "@/lib/types";

type SearchMode = "web" | "urls" | "both";

export default function WebSearchPage() {
  const [query, setQuery] = useState("");
  const [urlsInput, setUrlsInput] = useState("");
  const [mode, setMode] = useState<SearchMode>("web");
  const [type, setType] = useState("auto");
  const [count, setCount] = useState(10);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [response, setResponse] = useState<WebSearchResponse | null>(null);
  const [expandedUrl, setExpandedUrl] = useState<string | null>(null);
  const [expandedContent, setExpandedContent] = useState<string | null>(null);

  const handleSearch = useCallback(async () => {
    const q = query.trim();
    const urls = urlsInput
      .split("\n")
      .map((u) => u.trim())
      .filter((u) => u.length > 0);

    if (!q && urls.length === 0) return;

    setLoading(true);
    setError(null);
    setResponse(null);
    setExpandedUrl(null);
    setExpandedContent(null);

    try {
      const params: any = { type, count };
      if (mode === "web" || mode === "both") params.query = q;
      if (mode === "urls" || mode === "both") params.urls = urls;
      if (mode === "urls" && !q) delete params.query;

      const res = await webSearch(params);
      setResponse(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Search failed");
    } finally {
      setLoading(false);
    }
  }, [query, urlsInput, mode, type, count]);

  return (
    <div className="min-h-screen bg-surface-950 text-surface-50">
      <header className="sticky top-0 z-30 border-b border-white/8 bg-surface-950/80 backdrop-blur-xl">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4 sm:px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl border border-cyan-400/20 bg-cyan-400/10">
              <Globe className="h-5 w-5 text-cyan-300" />
            </div>
            <div>
              <h1 className="text-base font-semibold text-surface-50">Web Search</h1>
              <p className="text-xs text-surface-400">Multi-engine with auto-fallback</p>
            </div>
          </div>
          <a
            href="/"
            className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-surface-900/70 px-4 py-2 text-xs font-medium text-surface-300 transition-colors hover:border-cyan-400/20 hover:text-cyan-100"
          >
            <Brain className="h-3.5 w-3.5" />
            OSINT Search
          </a>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 pb-16 pt-8 sm:px-6">
        <div className="rounded-2xl border border-white/10 bg-surface-900/60 p-6 shadow-lg backdrop-blur-xl">
          <div className="mb-5 flex flex-wrap gap-2">
            {(["web", "urls", "both"] as const).map((m) => (
              <button
                key={m}
                onClick={() => setMode(m)}
                className={`rounded-full px-4 py-1.5 text-xs font-medium transition-colors ${
                  mode === m
                    ? "bg-cyan-400/20 text-cyan-200 border border-cyan-400/30"
                    : "bg-surface-800/60 text-surface-400 border border-surface-700/60 hover:border-surface-600"
                }`}
              >
                {m === "web" ? "Search" : m === "urls" ? "Fetch URLs" : "Both"}
              </button>
            ))}
          </div>

          {(mode === "web" || mode === "both") && (
            <div className="mb-4">
              <label className="mb-1.5 block text-xs font-medium text-surface-400">Search query</label>
              <div className="relative">
                <Search className="pointer-events-none absolute left-3.5 top-3.5 h-4 w-4 text-surface-500" />
                <input
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Enter search query..."
                  className="w-full rounded-xl border border-surface-700/80 bg-surface-950/80 py-3 pl-10 pr-4 text-sm text-surface-50 placeholder-surface-500 focus:border-cyan-400 focus:outline-none focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
            </div>
          )}

          {(mode === "urls" || mode === "both") && (
            <div className="mb-4">
              <label className="mb-1.5 block text-xs font-medium text-surface-400">
                URLs to fetch (one per line)
              </label>
              <div className="relative">
                <Link className="pointer-events-none absolute left-3.5 top-3.5 h-4 w-4 text-surface-500" />
                <textarea
                  value={urlsInput}
                  onChange={(e) => setUrlsInput(e.target.value)}
                  placeholder="https://example.com&#10;https://example.org/page"
                  rows={3}
                  className="w-full rounded-xl border border-surface-700/80 bg-surface-950/80 py-3 pl-10 pr-4 text-sm text-surface-50 placeholder-surface-500 focus:border-cyan-400 focus:outline-none focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
            </div>
          )}

          <div className="mb-5 flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2">
              <label className="text-xs text-surface-400">Type:</label>
              <select
                value={type}
                onChange={(e) => setType(e.target.value)}
                className="rounded-lg border border-surface-700/80 bg-surface-950/80 px-2.5 py-1.5 text-xs text-surface-300 focus:border-cyan-400 focus:outline-none"
              >
                <option value="auto">Auto</option>
                <option value="fast">Fast</option>
                <option value="deep">Deep</option>
              </select>
            </div>
            <div className="flex items-center gap-2">
              <label className="text-xs text-surface-400">Results:</label>
              <select
                value={count}
                onChange={(e) => setCount(Number(e.target.value))}
                className="rounded-lg border border-surface-700/80 bg-surface-950/80 px-2.5 py-1.5 text-xs text-surface-300 focus:border-cyan-400 focus:outline-none"
              >
                {[5, 10, 15, 20, 30, 50].map((n) => (
                  <option key={n} value={n}>{n}</option>
                ))}
              </select>
            </div>
          </div>

          <button
            onClick={handleSearch}
            disabled={loading || (query.trim() === "" && urlsInput.trim() === "")}
            className="inline-flex w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-cyan-500 to-blue-500 px-5 py-3 text-sm font-semibold text-slate-950 shadow-lg transition-all hover:brightness-110 disabled:cursor-not-allowed disabled:from-surface-700 disabled:to-surface-700 disabled:text-surface-400"
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Globe className="h-4 w-4" />
            )}
            <span>{loading ? "Searching..." : mode === "urls" ? "Fetch URLs" : "Search Web"}</span>
          </button>
        </div>

        {error && (
          <div className="mt-6 flex items-start gap-3 rounded-xl border border-red-400/20 bg-red-950/30 p-4">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-red-400" />
            <p className="text-sm text-red-300">{error}</p>
          </div>
        )}

        {response && (
          <div className="mt-8 space-y-6">
            {response.sources.length > 0 && (
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs text-surface-500">Sources:</span>
                {response.sources.map((s) => (
                  <span
                    key={s}
                    className="rounded-full border border-white/8 bg-surface-800/60 px-3 py-1 text-xs text-surface-300"
                  >
                    {s}
                  </span>
                ))}
                <span className="ml-auto text-xs text-surface-500">
                  {response.total} result{response.total !== 1 ? "s" : ""}
                </span>
              </div>
            )}

            {response.results.length > 0 && (
              <div className="space-y-3">
                <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-[0.2em] text-surface-400">
                  <Search className="h-4 w-4" />
                  Search Results
                </h2>
                <div className="grid gap-3">
                  {response.results.map((r, i) => (
                    <div
                      key={`${r.url}-${i}`}
                      className="rounded-xl border border-white/8 bg-surface-900/50 p-4 transition-colors hover:border-white/15"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0 flex-1">
                          <a
                            href={r.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1.5 text-sm font-medium text-cyan-300 hover:text-cyan-200 hover:underline"
                          >
                            {r.title}
                            <ExternalLink className="h-3 w-3 shrink-0" />
                          </a>
                          <p className="mt-1 text-xs leading-5 text-surface-400">{r.description}</p>
                          <div className="mt-2 flex items-center gap-2">
                            <span className="truncate text-xs text-surface-500">{r.url}</span>
                            <span className="rounded bg-surface-800/80 px-2 py-0.5 text-[10px] font-medium text-surface-400">
                              {r.engine}
                            </span>
                          </div>
                        </div>
                        <span className="shrink-0 text-xs text-surface-500">#{r.rank}</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {response.pages.length > 0 && (
              <div className="space-y-3">
                <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-[0.2em] text-surface-400">
                  <FileText className="h-4 w-4" />
                  Fetched Pages
                </h2>
                <div className="grid gap-3">
                  {response.pages.map((p, i) => (
                    <div
                      key={`${p.url}-${i}`}
                      className="rounded-xl border border-white/8 bg-surface-900/50 transition-colors hover:border-white/15"
                    >
                      <button
                        onClick={() =>
                          setExpandedUrl(expandedUrl === p.url ? null : p.url)
                        }
                        className="flex w-full items-center justify-between gap-3 p-4 text-left"
                      >
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-1.5 text-sm font-medium text-amber-300">
                            <Link className="h-3.5 w-3.5 shrink-0" />
                            <span className="truncate">{p.title || p.url}</span>
                          </div>
                          <p className="mt-0.5 truncate text-xs text-surface-500">{p.url}</p>
                        </div>
                        <div className="flex shrink-0 items-center gap-2">
                          <span className="text-xs text-surface-500">HTTP {p.status_code}</span>
                          {expandedUrl === p.url ? (
                            <ChevronUp className="h-4 w-4 text-surface-400" />
                          ) : (
                            <ChevronDown className="h-4 w-4 text-surface-400" />
                          )}
                        </div>
                      </button>
                      {expandedUrl === p.url && p.content && (
                        <div className="border-t border-white/8 px-4 pb-4">
                          <pre className="mt-3 max-h-96 overflow-auto rounded-lg bg-surface-950/80 p-3 text-xs leading-6 text-surface-300">
                            {p.content.slice(0, 5000)}
                            {p.content.length > 5000 && "..."}
                          </pre>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {response.total === 0 && (
              <div className="rounded-xl border border-white/8 bg-surface-900/40 p-8 text-center">
                <AlertCircle className="mx-auto h-8 w-8 text-surface-500" />
                <p className="mt-3 text-sm text-surface-400">No results returned.</p>
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}

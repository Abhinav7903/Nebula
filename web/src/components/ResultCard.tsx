"use client";

import { useState } from "react";
import { ExternalLink, ChevronDown, ChevronUp, Tag, Shield, Database, Sparkles } from "lucide-react";
import type { Result } from "@/lib/types";

const BG_COLORS: Record<string, string> = {
  shodan: "from-red-500/15 via-surface-900/80 to-surface-900/40 border-red-400/20",
  censys: "from-blue-500/15 via-surface-900/80 to-surface-900/40 border-blue-400/20",
  virustotal: "from-orange-500/15 via-surface-900/80 to-surface-900/40 border-orange-400/20",
  greynoise: "from-purple-500/15 via-surface-900/80 to-surface-900/40 border-purple-400/20",
  github: "from-slate-500/15 via-surface-900/80 to-surface-900/40 border-slate-400/20",
  urlscan: "from-yellow-500/15 via-surface-900/80 to-surface-900/40 border-yellow-400/20",
  emailrep: "from-pink-500/15 via-surface-900/80 to-surface-900/40 border-pink-400/20",
  dns: "from-emerald-500/15 via-surface-900/80 to-surface-900/40 border-emerald-400/20",
  whois: "from-teal-500/15 via-surface-900/80 to-surface-900/40 border-teal-400/20",
  crtsh: "from-indigo-500/15 via-surface-900/80 to-surface-900/40 border-indigo-400/20",
  subdomains: "from-sky-500/15 via-surface-900/80 to-surface-900/40 border-sky-400/20",
  dnsdumpster: "from-violet-500/15 via-surface-900/80 to-surface-900/40 border-violet-400/20",
  geoip: "from-lime-500/15 via-surface-900/80 to-surface-900/40 border-lime-400/20",
  ethereum: "from-cyan-500/15 via-surface-900/80 to-surface-900/40 border-cyan-400/20",
  bitcoin: "from-amber-500/15 via-surface-900/80 to-surface-900/40 border-amber-400/20",
  solana: "from-fuchsia-500/15 via-surface-900/80 to-surface-900/40 border-fuchsia-400/20",
  searchengine: "from-rose-500/15 via-surface-900/80 to-surface-900/40 border-rose-400/20",
  wayback: "from-stone-500/15 via-surface-900/80 to-surface-900/40 border-stone-400/20",
  pastebin: "from-red-400/15 via-surface-900/80 to-surface-900/40 border-red-300/20",
  social: "from-blue-400/15 via-surface-900/80 to-surface-900/40 border-blue-300/20",
  threatintel: "from-orange-400/15 via-surface-900/80 to-surface-900/40 border-orange-300/20",
  onion: "from-green-800/15 via-surface-900/80 to-surface-900/40 border-green-700/20",
};

const COLLECTOR_LABELS: Record<string, string> = {
  shodan: "Shodan",
  censys: "Censys",
  virustotal: "VirusTotal",
  greynoise: "GreyNoise",
  github: "GitHub",
  urlscan: "URLScan",
  emailrep: "EmailRep",
  dns: "DNS",
  whois: "WHOIS",
  crtsh: "crt.sh",
  subdomains: "Subdomains",
  dnsdumpster: "DNS Dumpster",
  geoip: "GeoIP",
  ethereum: "Ethereum",
  bitcoin: "Bitcoin",
  solana: "Solana",
  searchengine: "Search Engine",
  wayback: "Wayback Machine",
  pastebin: "Pastebin",
  social: "Social Media",
  threatintel: "Threat Intel",
  onion: "Onion",
};

function confidenceTone(c: number): string {
  if (c >= 0.8) return "text-emerald-300 bg-emerald-500/10 border-emerald-400/20";
  if (c >= 0.5) return "text-amber-300 bg-amber-500/10 border-amber-400/20";
  return "text-rose-300 bg-rose-500/10 border-rose-400/20";
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString();
}

function isDomain(s: string): boolean {
  return s.includes(".");
}

function sourceHref(source: string): string {
  return `https://${source}`;
}

interface Props {
  result: Result;
}

export default function ResultCard({ result }: Props) {
  const [expanded, setExpanded] = useState(false);
  const accent = BG_COLORS[result.collector] || "from-surface-600/15 via-surface-900/80 to-surface-900/40 border-surface-700/40";
  const confidence = (result.confidence * 100).toFixed(0);

  return (
    <article
      role="button"
      tabIndex={0}
      onClick={() => setExpanded((current) => !current)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          setExpanded((current) => !current);
        }
      }}
      className={`group relative flex h-full min-h-[24rem] flex-col overflow-hidden rounded-3xl border bg-gradient-to-br ${accent} shadow-[0_18px_50px_rgba(2,6,23,0.35)] transition-all duration-300 hover:border-white/15 hover:shadow-[0_24px_70px_rgba(2,6,23,0.55)]`}
    >
      <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-white/30 to-transparent" />
      <div className="absolute right-0 top-0 h-24 w-24 rounded-full bg-cyan-400/10 blur-2xl transition-transform duration-500 group-hover:scale-125" />

      <div className="relative flex h-full flex-1 flex-col p-5 sm:p-6">
        <div className="mb-4 flex items-start justify-between gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-surface-950/55 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.2em] text-surface-200">
              <Sparkles className="h-3.5 w-3.5 text-cyan-300" />
              {COLLECTOR_LABELS[result.collector] || result.collector}
            </span>
            <span className="rounded-full border border-surface-700/80 bg-surface-900/70 px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em] text-surface-400">
              {result.type}
            </span>
          </div>
          {result.confidence > 0 && (
            <span className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold ${confidenceTone(result.confidence)}`}>
              <Shield className="h-3.5 w-3.5" />
              {confidence}%
            </span>
          )}
        </div>

        <h3 className="text-lg font-semibold leading-snug text-surface-50 sm:text-[1.15rem]">
          {result.title}
        </h3>

        {result.description && (
          <p className="mt-2 line-clamp-3 text-sm leading-6 text-surface-300">
            {result.description}
          </p>
        )}

        <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-surface-400">
          {result.url && (
            <a
              href={result.url}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              className="inline-flex items-center gap-1.5 rounded-full border border-cyan-400/20 bg-cyan-400/8 px-3 py-1.5 text-cyan-200 transition-colors hover:bg-cyan-400/15 hover:text-cyan-100"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              Open source
            </a>
          )}
          {isDomain(result.source) ? (
            <a
              href={sourceHref(result.source)}
              target="_blank"
              rel="noopener noreferrer"
              onClick={(e) => e.stopPropagation()}
              className="inline-flex items-center gap-1.5 transition-colors hover:text-surface-200"
            >
              <Database className="h-3.5 w-3.5" />
              {result.source}
            </a>
          ) : (
            <span className="inline-flex items-center gap-1.5">
              <Database className="h-3.5 w-3.5" />
              {result.source}
            </span>
          )}
          <span>{formatTime(result.found_at)}</span>
        </div>

        {result.tags.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2">
            {result.tags.map((t, i) => (
              <span
                key={`${t}-${i}`}
                className="inline-flex items-center gap-1 rounded-full border border-surface-700/80 bg-surface-950/50 px-2.5 py-1 text-xs text-surface-300"
              >
                <Tag className="h-3 w-3 text-surface-500" />
                {t}
              </span>
            ))}
          </div>
        )}

        {result.data && Object.keys(result.data).length > 0 && (
          <div className="mt-auto pt-4">
            <button
              onClick={(e) => {
                e.stopPropagation();
                setExpanded((current) => !current);
              }}
              className="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-surface-950/60 px-3 py-1.5 text-xs font-medium text-surface-300 transition-colors hover:border-white/20 hover:text-surface-100"
            >
              {expanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
              {expanded ? "Hide details" : "Show details"}
            </button>
            {expanded && (
              <pre className="mt-3 max-h-32 overflow-auto rounded-2xl border border-surface-700/70 bg-surface-950/80 p-4 text-xs leading-6 text-surface-300 shadow-inner shadow-black/30">
                {JSON.stringify(result.data, null, 2)}
              </pre>
            )}
          </div>
        )}
      </div>
    </article>
  );
}

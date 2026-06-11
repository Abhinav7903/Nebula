export interface Result {
  id: string;
  collector: string;
  type: string;
  title: string;
  description: string;
  url?: string;
  data: Record<string, any>;
  tags: string[];
  confidence: number;
  source: string;
  found_at: string;
}

export interface Stats {
  collectors_run: number;
  collectors_ok: number;
  collectors_failed: number;
  results_found: number;
  duration_ms: number;
}

export interface Search {
  search_id: string;
  query: string;
  query_type: string;
  status: "pending" | "running" | "done" | "failed" | "cancelled";
  started_at: string;
  finished_at?: string;
  summary: string;
  stats: Stats;
  results: Result[];
}

export interface SearchSummary {
  search_id: string;
  query: string;
  query_type: string;
  status: string;
  started_at: string;
}

export interface SSEEvent {
  event: string;
  payload: any;
}

import type { Search, SearchSummary } from "./types";

const BASE = "";

export async function createSearch(query: string): Promise<{ search_id: string; query_type: string }> {
  const res = await fetch(`${BASE}/api/v1/search`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ detail: "Request failed" }));
    throw new Error(err.detail || `HTTP ${res.status}`);
  }
  return res.json();
}

export async function getSearch(id: string): Promise<Search> {
  const res = await fetch(`${BASE}/api/v1/search/${id}`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function listSearches(): Promise<SearchSummary[]> {
  const res = await fetch(`${BASE}/api/v1/searches`);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

export async function deleteSearch(id: string): Promise<void> {
  await fetch(`${BASE}/api/v1/search/${id}`, { method: "DELETE" });
}

export function streamSearch(
  id: string,
  onEvent: (event: string, payload: any) => void,
  onError: (err: Error) => void
): () => void {
  const url = `${BASE}/api/v1/search/${id}/stream`;
  const es = new EventSource(url);

  es.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data);
      onEvent(data.event || "message", data.payload || data);
    } catch {
      onEvent("message", e.data);
    }
  };

  es.onerror = () => {
    onError(new Error("SSE connection failed"));
    es.close();
  };

  return () => es.close();
}

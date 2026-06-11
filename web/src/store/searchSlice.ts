import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import type { Result, Search, SearchSummary } from "@/lib/types";

export interface SearchState {
  activeId: string | null;
  activeSearch: Search | null;
  liveResults: Result[];
  recentSearches: SearchSummary[];
  loading: boolean;
}

const initialState: SearchState = {
  activeId: null,
  activeSearch: null,
  liveResults: [],
  recentSearches: [],
  loading: false,
};

function mergeResults(current: Result[], incoming: Result[]): Result[] {
  const seen = new Map<string, Result>();

  for (const result of current) {
    seen.set(result.id, result);
  }

  for (const result of incoming) {
    seen.set(result.id, result);
  }

  return Array.from(seen.values());
}

function sortSearches(searches: SearchSummary[]): SearchSummary[] {
  return [...searches].sort(
    (a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
  );
}

const searchSlice = createSlice({
  name: "search",
  initialState,
  reducers: {
    beginSearch(state) {
      state.loading = true;
      state.activeId = null;
      state.activeSearch = null;
      state.liveResults = [];
    },
    setLoading(state, action: PayloadAction<boolean>) {
      state.loading = action.payload;
    },
    setActiveId(state, action: PayloadAction<string | null>) {
      state.activeId = action.payload;
    },
    setActiveSearch(state, action: PayloadAction<Search | null>) {
      state.activeSearch = action.payload;
    },
    clearActiveSearch(state) {
      state.activeId = null;
      state.activeSearch = null;
      state.liveResults = [];
      state.loading = false;
    },
    setRecentSearches(state, action: PayloadAction<SearchSummary[]>) {
      state.recentSearches = sortSearches(action.payload);
    },
    removeRecentSearch(state, action: PayloadAction<string>) {
      state.recentSearches = state.recentSearches.filter(
        (search) => search.search_id !== action.payload
      );
      if (state.activeId === action.payload) {
        state.activeId = null;
        state.activeSearch = null;
        state.liveResults = [];
        state.loading = false;
      }
    },
    mergeLiveResults(state, action: PayloadAction<Result[]>) {
      state.liveResults = mergeResults(state.liveResults, action.payload);
    },
    replaceLiveResults(state, action: PayloadAction<Result[]>) {
      state.liveResults = mergeResults([], action.payload);
    },
    clearLiveResults(state) {
      state.liveResults = [];
    },
  },
});

export const {
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
} = searchSlice.actions;

export default searchSlice.reducer;

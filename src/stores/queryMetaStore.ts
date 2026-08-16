import { create } from "zustand";
import type { QueryMeta } from "../lib/api/queryClient";

export interface QueryMetaEntry extends QueryMeta {
  at: number;
  query: string;
}

interface QueryMetaState {
  /** Most recent query meta per cache scope (e.g. "default", "explore"). */
  lastByScope: Record<string, QueryMetaEntry>;
  /** Meta of the most recent query overall, regardless of scope. */
  last: QueryMetaEntry | null;
  record: (scope: string, query: string, meta: QueryMeta) => void;
}

export const useQueryMetaStore = create<QueryMetaState>((set) => ({
  lastByScope: {},
  last: null,
  record: (scope, query, meta) => {
    const entry: QueryMetaEntry = { ...meta, at: Date.now(), query };
    set((state) => ({
      last: entry,
      lastByScope: { ...state.lastByScope, [scope]: entry },
    }));
  },
}));

/** Imperative recorder usable outside React (SWR fetchers). */
export const recordQueryMeta = (
  scope: string,
  query: string,
  meta: QueryMeta,
) => {
  useQueryMetaStore.getState().record(scope, query, meta);
};

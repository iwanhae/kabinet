export interface InvestigationConfig {
  openaiApiKey: string;
  openaiApiBase: string;
  openaiModel?: string;
  kubeApiUrl: string;
}

/** One run_sql tool call — a numbered piece of evidence in the case file. */
export interface Exhibit {
  id: string;
  seq: number;
  sql: string;
  start: string;
  end: string;
  status: "running" | "done" | "error";
  rowCount?: number;
  durationMs?: number;
  /** Preview rows (capped) for display. */
  rows?: Record<string, unknown>[];
  error?: string;
}

export interface ExhibitChart {
  title: string;
  type: "bar" | "line";
  content: Record<string, unknown>[];
}

export interface CaseTurn {
  id: string;
  role: "user" | "assistant";
  /** Markdown; streams in for assistant turns. */
  text: string;
  exhibits: Exhibit[];
  charts: ExhibitChart[];
  error?: string;
}

export interface CaseSession {
  id: string;
  timestamp: number;
  title: string;
  turns: CaseTurn[];
}

export type InvestigationStatus =
  | "idle"
  | "thinking"
  | "querying"
  | "streaming"
  | "complete"
  | "error";

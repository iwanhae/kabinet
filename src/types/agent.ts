export interface Message {
  role: "system" | "user" | "assistant";
  content: string;
}

export interface InvestigationConfig {
  openaiApiKey: string;
  openaiApiBase: string;
  openaiModel?: string;
  kubeApiUrl: string;
}

export interface QueryResult {
  results?: any[];
  error?: string;
}

export interface AgentPlan {
  thought?: string;
  hypothesis?: string;
  query?: {
    sql: string;
    start: string;
    end: string;
  };
  final_analysis?: string;
  data?: {
    type: "table" | "bar_chart" | "line_chart";
    title: string;
    content: any;
  };
}

export type InvestigationStatus =
  | "idle"
  | "planning"
  | "querying"
  | "analyzing"
  | "complete"
  | "error";

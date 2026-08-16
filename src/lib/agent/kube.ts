import type { InvestigationConfig } from "../../types/agent";

export interface KubeQueryResult {
  results?: Record<string, unknown>[];
  duration_ms?: number;
  error?: string;
}

export const executeKubeQuery = async (
  config: InvestigationConfig,
  query: string,
  start: string,
  end: string,
): Promise<KubeQueryResult> => {
  try {
    const response = await fetch(config.kubeApiUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query, start, end }),
    });

    if (!response.ok) {
      const text = await response.text();
      return { error: `API call failed: ${response.status} ${text}` };
    }

    return (await response.json()) as KubeQueryResult;
  } catch (e) {
    const errorMessage = e instanceof Error ? e.message : "Unknown error";
    return { error: `Network error: ${errorMessage}` };
  }
};

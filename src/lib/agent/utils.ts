import type { QueryResult } from "../../types/agent";

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === "object" && value !== null && !Array.isArray(value);
};

export const summarizeResult = (result: QueryResult): string => {
  if (result.error) {
    return `Query failed with error: ${result.error}`;
  }

  const resultsList = result.results || [];
  if (resultsList.length === 0) {
    return "Query returned no results (an empty list: []).";
  }

  const count = resultsList.length;
  const firstRow = resultsList[0];
  const columns = isRecord(firstRow)
    ? Object.keys(firstRow).join(", ")
    : "unknown";

  let summary = `Query returned ${count} rows. Columns: ${columns}. `;
  if (count > 0) {
    summary += `First row summary: ${JSON.stringify(firstRow)}`;
  }

  return summary;
};

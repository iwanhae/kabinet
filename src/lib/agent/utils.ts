import type { QueryResult } from "../../types/agent";

export const summarizeResult = (result: QueryResult): string => {
  if (result.error) {
    return `Query failed with error: ${result.error}`;
  }

  const resultsList = result.results || [];
  if (resultsList.length === 0) {
    return "Query returned no results (an empty list: []).";
  }

  const count = resultsList.length;
  const columns = Object.keys(resultsList[0]).join(", ");

  let summary = `Query returned ${count} rows. Columns: ${columns}. `;
  if (count > 0) {
    summary += `First row summary: ${JSON.stringify(resultsList[0])}`;
  }

  return summary;
};

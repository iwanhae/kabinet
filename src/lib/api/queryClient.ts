/**
 * Thin client for the backend query API.
 * Returns both the result rows and the scan-cost metadata the server reports.
 */

export interface QueryMeta {
  duration_ms: number;
  files: { path: string; size: number }[];
  total_files_size_bytes: number;
}

export interface QueryResponse<T> {
  results: T[];
  meta: QueryMeta;
}

interface ApiErrorResponse {
  error: string;
}

const isErrorResponse = (response: unknown): response is ApiErrorResponse => {
  return (
    typeof response === "object" &&
    response !== null &&
    "error" in response &&
    typeof (response as ApiErrorResponse).error === "string"
  );
};

export async function postQuery<T>(
  query: string,
  from: string,
  to: string,
  signal?: AbortSignal,
): Promise<QueryResponse<T>> {
  const response = await fetch("/query", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, start: from, end: to }),
    signal,
  });

  if (!response.ok) {
    const body = await response.text();
    throw new Error(`An error occurred while fetching the data: ${body}`);
  }

  const data: unknown = await response.json();

  if (isErrorResponse(data)) {
    throw new Error(data.error);
  }

  const payload = data as {
    results: T[] | null;
    duration_ms: number;
    files: { path: string; size: number }[] | null;
    total_files_size_bytes: number;
  };

  return {
    results: payload.results ?? [],
    meta: {
      duration_ms: payload.duration_ms ?? 0,
      files: payload.files ?? [],
      total_files_size_bytes: payload.total_files_size_bytes ?? 0,
    },
  };
}

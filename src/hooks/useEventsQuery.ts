import useSWR, { mutate, type SWRResponse } from "swr";
import { useTimeRangeStore } from "../stores/timeRangeStore";

/**
 * Represents the structure of the API response for a successful query.
 */
interface ApiResponse<T> {
  results: T[];
}

/**
 * The expected structure of an error response from the API.
 */
interface ApiErrorResponse {
  error: string;
}

/**
 * A generic type guard to check if the response is an error response.
 * @param response The response object from the fetch call.
 * @returns True if the response is an ApiErrorResponse.
 */
const isErrorResponse = (response: unknown): response is ApiErrorResponse => {
  return (
    typeof response === "object" &&
    response !== null &&
    "error" in response &&
    typeof (response as ApiErrorResponse).error === "string"
  );
};

/**
 * The fetcher function for SWR. It sends a POST request to the query API.
 * @param key An array containing the API endpoint, query string, start time, and end time.
 * @returns A promise that resolves to the array of query results.
 * @throws An error if the fetch fails or the API returns an error.
 */
const fetcher = async <T>(
  key: [string, string, string, string],
): Promise<T[]> => {
  const [, query, from, to] = key;

  const response = await fetch("/query", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ query, start: from, end: to }),
  });

  if (!response.ok) {
    throw new Error(
      `An error occurred while fetching the data: ${response.statusText}`,
    );
  }

  const data: ApiResponse<T> | ApiErrorResponse = await response.json();

  if (isErrorResponse(data)) {
    throw new Error(data.error);
  }

  return data.results;
};

/**
 * A generic custom hook to query the events API using SWR.
 * It automatically uses the global time range from the Zustand store.
 *
 * @param query The SQL query string to execute. If null, the request will not be sent.
 * @returns The SWR response object, which includes `data`, `error`, and `isLoading` states.
 *
 * @example
 * ```tsx
 * interface ReasonCount {
 *   reason: string;
 *   count: number;
 * }
 *
 * const { data, error, isLoading } = useEventsQuery<ReasonCount>("SELECT reason, COUNT(*) as count FROM $events GROUP BY reason");
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const useEventsQuery = <T extends Record<string, any>>(
  query: string | null,
): SWRResponse<T[], Error> => {
  const { from, to } = useTimeRangeStore();

  const key = query ? ["/events", query, from, to] : null;

  return useSWR<T[], Error>(key, fetcher, {
    // Re-fetch on window focus can be helpful but also aggressive.
    // Let's disable it for now to prevent unnecessary API calls.
    revalidateOnFocus: false,
  });
};

export const invalidateEventsQuery = () => {
  mutate(() => true, undefined, { revalidate: true });
};

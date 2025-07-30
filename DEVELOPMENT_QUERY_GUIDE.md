# Kube Event Analyzer - Querying Guide

This guide provides instructions and examples on how to query the Kube Event Analyzer API.

## API Endpoint

- **URL**: `http://localhost:8080/query`
- **Method**: `POST`
- **Content-Type**: `application/json`

## Query Payload Structure

The API expects a JSON payload with the following fields:

```json
{
  "query": "YOUR_SQL_QUERY",
  "start": "START_TIME_ISO_8601",
  "end": "END_TIME_ISO_8601"
}
```

- `query`: The SQL query string.
- `start` / `end`: These parameters are crucial for performance. They define the time range of data to be scanned, preventing the need to query the entire multi-gigabyte dataset. Always specify the narrowest possible time range.

## API Response

The API returns a JSON object containing a `results` field.

- `results`: An array of JSON objects, where each object represents a row in the query result. The keys of the objects correspond to the column names in your `SELECT` statement.

**Example Response:**

For a query like `SELECT reason, COUNT(*) as count FROM $events ...`, the response will look like this:

```json
{
  "results": [
    {
      "reason": "FailedMount",
      "count": 26421
    },
    {
      "reason": "Unhealthy",
      "count": 21502
    }
  ]
}
```

In case of a query error, the response will contain an error message instead:

```json
{
  "error": "Binder Error: Referenced column \"kube-system\" not found..."
}
```

## The `$events` Macro

In your SQL queries, you don't query a traditional table name. Instead, you use the `$events` macro. This macro dynamically represents all the Kubernetes events within the specified `start` and `end` time range, combining both real-time in-memory data and historical data from Parquet files.

**Example:**

```sql
SELECT * FROM $events LIMIT 10
```

## Schema and Nested Fields

The `$events` table has a nested structure. To access fields within a `STRUCT`, use dot notation.

For example, to access the `namespace` of an event, you use `metadata.namespace`. The same applies to other nested fields like `involvedObject.name`.

Refer to the `README.md` for the full table schema.

**Example: Grouping by namespace**

```sql
SELECT metadata.namespace, COUNT(*) as count
FROM $events
GROUP BY metadata.namespace
ORDER BY count DESC
```

## Writing Queries with `curl`

When using `curl` to send queries from the command line, you need to be careful with quotes. The shell can interpret and modify quotes in your JSON payload, leading to SQL errors.

### The Problem with Quotes

A common mistake is to write a query where the single quotes required for SQL string literals conflict with the single quotes used by the shell.

```shell
# This will likely FAIL because the shell mishandles the inner quotes.
curl -X POST http://localhost:8080/query \
-H "Content-Type: application/json" \
-d '{
    "query": "SELECT reason, COUNT(*) FROM $events WHERE type = 'Warning' GROUP BY reason",
    "start": "2025-01-01T00:00:00Z",
    "end": "2026-01-02T00:00:00Z"
}'
```

The server would receive a malformed query because the shell interprets `'Warning'` incorrectly.

### The Recommended Solution: Using a "Here Document"

The most reliable way to send complex JSON payloads with `curl` is to use a "here document". This method prevents the shell from interfering with the quotes in your JSON string, sending it to the server exactly as written.

**Syntax:**

```shell
curl [options] -d @- <<'EOF'
{
  "json": "payload"
}
EOF
```

**Example:**
This is the correct and safest way to run a query with string literals.

```shell
curl -X POST http://localhost:8080/query -H "Content-Type: application/json" -d @- <<'EOF'
{
    "query": "SELECT reason, COUNT(*) as count FROM $events WHERE metadata.namespace = 'kube-system' AND type = 'Warning' GROUP BY reason ORDER BY count DESC LIMIT 10",
    "start": "2025-01-01T00:00:00Z",
    "end": "2026-01-02T00:00:00Z"
}
EOF
```

Notice the single quotes around `'kube-system'` and `'Warning'` are preserved, ensuring the SQL query is valid.

## Temporal Analysis

For time-based analysis, it is essential to use the correct timestamp field and appropriate functions.

### Use `metadata.creationTimestamp`

The primary timestamp for events is `metadata.creationTimestamp`. The `eventTime` field is deprecated and may contain null or incorrect values, making it unreliable for temporal queries.

**Always use `metadata.creationTimestamp` for any time-based analysis.**

### Time-Windowing Functions

DuckDB provides powerful functions for creating time windows, which are perfect for analyzing trends.

- `date_trunc('unit', timestamp)`: Truncates a timestamp to a specified unit (e.g., 'hour', 'day'). This is useful for creating fixed-size, non-overlapping (tumbling) windows.
- `time_bucket(interval, timestamp)`: A more flexible function that buckets a timestamp into a specified interval (e.g., `INTERVAL 15 MINUTE`).

**Example: Hourly event count**

```sql
SELECT
    date_trunc('hour', metadata.creationTimestamp) AS hour,
    COUNT(*) AS count
FROM $events
WHERE reason = 'Scheduled'
GROUP BY hour
ORDER BY hour
```

**Example: 15-minute warning event breakdown**

```sql
SELECT
    time_bucket(INTERVAL 15 MINUTE, metadata.creationTimestamp) AS bucket,
    reason,
    COUNT(*) AS count
FROM $events
WHERE type = 'Warning'
GROUP BY bucket, reason
ORDER BY bucket, count DESC
```

## Common Query Examples

### Count events by type

```sql
SELECT type, COUNT(*) as count
FROM $events
GROUP BY type
ORDER BY count DESC
```

### Top 10 warning reasons in a specific namespace

```sql
SELECT reason, COUNT(*) as count
FROM $events
WHERE metadata.namespace = 'some-namespace' AND type = 'Warning'
GROUP BY reason
ORDER BY count DESC
LIMIT 10
```

### Find pods with a specific warning reason

```sql
SELECT involvedObject.name, COUNT(*) as count
FROM $events
WHERE reason = 'FailedMount'
GROUP BY involvedObject.name
ORDER BY count DESC
LIMIT 10
```

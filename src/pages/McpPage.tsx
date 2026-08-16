import React from "react";
import styles from "./McpPage.module.css";

const CodeBlock: React.FC<{ children: string }> = ({ children }) => (
  <pre className={styles.code}>
    <code>{children}</code>
  </pre>
);

const McpPage: React.FC = () => {
  const endpoint = `${window.location.origin}/mcp`;

  return (
    <div className={styles.page}>
      <div className={styles.hero}>
        <h1>MCP Server</h1>
        <p>
          Kabinet ships a built-in{" "}
          <a
            href="https://modelcontextprotocol.io"
            target="_blank"
            rel="noreferrer"
          >
            Model Context Protocol
          </a>{" "}
          server, so AI assistants like Claude can query this cluster&apos;s
          Kubernetes events directly with SQL. The transport is streamable HTTP
          and the server is stateless — no API key or session affinity required.
        </p>
      </div>

      <section className={styles.section}>
        <h2>Endpoint</h2>
        <div className={styles.endpoint}>
          <span className={styles.endpointUrl}>{endpoint}</span>
        </div>
        <p>
          The endpoint is served by this Kabinet instance. If AI assistants
          reach the server through a different host (e.g. an in-cluster
          Service), substitute that address.
        </p>
      </section>

      <section className={styles.section}>
        <h2>Connect a client</h2>

        <h3>Claude Code</h3>
        <CodeBlock>{`claude mcp add --transport http kabinet ${endpoint}`}</CodeBlock>

        <h3>JSON config (Cursor, VS Code, and other MCP clients)</h3>
        <CodeBlock>{`{
  "mcpServers": {
    "kabinet": {
      "type": "http",
      "url": "${endpoint}"
    }
  }
}`}</CodeBlock>

        <h3>Claude Desktop / claude.ai</h3>
        <p>
          Settings → Connectors → <em>Add custom connector</em>, then paste the
          endpoint URL above. Note that claude.ai connects from Anthropic&apos;s
          servers, so the endpoint must be reachable from the internet.
        </p>
      </section>

      <section className={styles.section}>
        <h2>Available tools</h2>
        <table className={styles.toolTable}>
          <thead>
            <tr>
              <th>Tool</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>query_events</td>
              <td>
                Runs a read-only DuckDB SQL query over the event store. Takes{" "}
                <code>query</code>, <code>start</code>, and <code>end</code>{" "}
                (RFC3339). Results are capped at 1,000 rows.
              </td>
            </tr>
            <tr>
              <td>get_stats</td>
              <td>
                Returns storage statistics: WAL ingest state, archive file
                counts, and total size.
              </td>
            </tr>
          </tbody>
        </table>
      </section>

      <section className={styles.section}>
        <h2>Query guide for AI assistants</h2>
        <p>
          The full guide below is also embedded in the <code>query_events</code>{" "}
          tool description, so any connected model receives it automatically —
          no extra prompting needed.
        </p>
        <ul>
          <li>
            Query <code>FROM $events</code> — a macro that expands to exactly
            the data files overlapping the <code>start</code>/<code>end</code>{" "}
            window. Always pass the narrowest window; it controls scan cost.
          </li>
          <li>
            Use <code>lastTimestamp</code> for all time filters and bucketing (
            <code>eventTime</code> is frequently NULL).
          </li>
          <li>
            Nested fields use dot notation: <code>metadata.namespace</code>,{" "}
            <code>involvedObject.name</code>, <code>source.host</code>.
          </li>
          <li>
            Aggregate first (<code>GROUP BY reason</code>,{" "}
            <code>GROUP BY metadata.namespace</code>), then drill into raw rows
            with a tight <code>WHERE</code> and a small <code>LIMIT</code>.
          </li>
          <li>
            <code>type</code> is <code>'Normal'</code> or <code>'Warning'</code>{" "}
            — warnings are where problems live.
          </li>
        </ul>

        <h3>Example</h3>
        <CodeBlock>{`SELECT time_bucket(INTERVAL 5 MINUTE, lastTimestamp) AS bucket,
       reason, COUNT(*) AS c
FROM $events
WHERE type = 'Warning' AND metadata.namespace = 'prod'
GROUP BY bucket, reason
ORDER BY bucket, c DESC`}</CodeBlock>
      </section>
    </div>
  );
};

export default McpPage;

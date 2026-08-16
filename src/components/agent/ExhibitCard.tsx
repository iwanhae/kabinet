import React from "react";
import type { Exhibit } from "../../types/agent";
import { Accordion, Spinner, cx } from "../../ui";
import { formatCount } from "../../utils/format";
import styles from "./ExhibitCard.module.css";

const RowsTable: React.FC<{ rows: Record<string, unknown>[] }> = ({ rows }) => {
  const columns = rows[0] ? Object.keys(rows[0]) : [];
  return (
    <div style={{ overflow: "auto", maxHeight: 280 }}>
      <table className={styles.table}>
        <thead>
          <tr>
            {columns.map((c) => (
              <th key={c}>{c}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i}>
              {columns.map((c) => (
                <td key={c} title={String(row[c] ?? "")}>
                  {String(row[c] ?? "")}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export const ExhibitCard: React.FC<{ exhibit: Exhibit }> = ({ exhibit }) => (
  <div
    className={cx(
      styles.exhibit,
      exhibit.status === "error" && styles.exhibitError,
    )}
  >
    <div className={styles.header}>
      <span className={styles.tag}>
        EXHIBIT {String(exhibit.seq).padStart(2, "0")}
      </span>
      {exhibit.status === "running" ? (
        <>
          <Spinner size={12} />
          <span>running…</span>
        </>
      ) : exhibit.status === "error" ? (
        <span>failed</span>
      ) : (
        <span>
          {formatCount(exhibit.rowCount ?? 0)} rows
          {exhibit.durationMs !== undefined && ` · ${exhibit.durationMs} ms`}
        </span>
      )}
    </div>
    <pre className={styles.sql}>{exhibit.sql.trim()}</pre>
    {exhibit.error && <div className={styles.error}>{exhibit.error}</div>}
    {exhibit.rows && exhibit.rows.length > 0 && (
      <div className={styles.resultWrap}>
        <Accordion summary={`Result preview (${exhibit.rows.length} rows)`}>
          <RowsTable rows={exhibit.rows} />
        </Accordion>
      </div>
    )}
  </div>
);

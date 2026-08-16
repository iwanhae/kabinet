import React, { useRef, useState } from "react";
import { useQueryMetaStore } from "../stores/queryMetaStore";
import { formatBytes } from "../utils/format";
import { Popover } from "../ui";
import styles from "./ScanCostBar.module.css";

/**
 * The ledger stamp: every query prints what it cost.
 * Shows the most recent query's scan metadata from the backend.
 */
const ScanCostBar: React.FC = () => {
  const last = useQueryMetaStore((s) => s.last);
  const anchorRef = useRef<HTMLButtonElement>(null);
  const [filesOpen, setFilesOpen] = useState(false);

  if (!last) {
    return (
      <footer className={styles.bar}>
        <span className={styles.stamp}>SCAN</span>
        <span className={styles.query}>no queries yet</span>
      </footer>
    );
  }

  return (
    <footer className={styles.bar}>
      <span className={styles.stamp}>SCAN</span>
      <span className={styles.query} title={last.query}>
        {last.query.replace(/\s+/g, " ").trim()}
      </span>
      <span className={styles.metric}>{last.duration_ms} ms</span>
      <button
        ref={anchorRef}
        type="button"
        className={styles.filesButton}
        onClick={() => setFilesOpen((v) => !v)}
        disabled={last.files.length === 0}
      >
        {last.files.length} files
      </button>
      <span className={styles.metric}>
        {formatBytes(last.total_files_size_bytes)}
      </span>

      <Popover
        open={filesOpen}
        anchorEl={anchorRef.current}
        onClose={() => setFilesOpen(false)}
        align="end"
      >
        <div className={styles.fileList}>
          {last.files.map((f) => (
            <div key={f.path} className={styles.fileRow}>
              <span className={styles.filePath} title={f.path}>
                {f.path}
              </span>
              <span className={styles.fileSize}>{formatBytes(f.size)}</span>
            </div>
          ))}
        </div>
      </Popover>
    </footer>
  );
};

export default ScanCostBar;

import React from "react";
import dayjs from "dayjs";
import { Trash2 } from "lucide-react";
import type { CaseSession } from "../../types/agent";
import { Drawer, IconButton } from "../../ui";
import styles from "./HistoryPanel.module.css";

interface Props {
  open: boolean;
  onClose: () => void;
  sessions: CaseSession[];
  onSelect: (session: CaseSession) => void;
  onDelete: (id: string) => void;
}

export const HistoryPanel: React.FC<Props> = ({
  open,
  onClose,
  sessions,
  onSelect,
  onDelete,
}) => (
  <Drawer open={open} onClose={onClose} title="Case history" width={380}>
    {sessions.length === 0 ? (
      <div className={styles.empty}>No saved cases yet</div>
    ) : (
      <div className={styles.list}>
        {sessions.map((session) => (
          <div key={session.id} className={styles.item}>
            <button
              type="button"
              className={styles.select}
              onClick={() => {
                onSelect(session);
                onClose();
              }}
            >
              <span className={styles.title}>{session.title}</span>
              <span className={styles.date}>
                {dayjs(session.timestamp).format("MMM D, HH:mm")} ·{" "}
                {session.turns.filter((t) => t.role === "user").length}{" "}
                statements
              </span>
            </button>
            <IconButton
              label="Delete case"
              size="sm"
              onClick={() => onDelete(session.id)}
            >
              <Trash2 size={14} />
            </IconButton>
          </div>
        ))}
      </div>
    )}
  </Drawer>
);

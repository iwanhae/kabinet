import React, { useState } from "react";
import { FilePlus2, History, Settings } from "lucide-react";
import { useSettings } from "../hooks/agent/useSettings";
import { useInvestigation } from "../hooks/agent/useInvestigation";
import { useHistory } from "../hooks/agent/useHistory";
import { useTimeRange } from "../hooks/useUrlParams";
import { CaseTranscript } from "../components/agent/CaseTranscript";
import { Composer } from "../components/agent/Composer";
import { HistoryPanel } from "../components/agent/HistoryPanel";
import { SettingsModal } from "../components/agent/SettingsModal";
import { IconButton } from "../ui";
import styles from "./AgentPage.module.css";

const AgentPage: React.FC = () => {
  const { config, saveConfig, isOpen, openSettings, closeSettings } =
    useSettings();
  const { turns, status, start, stop, clearSession, loadSession } =
    useInvestigation(config);
  const { sessions, deleteSession } = useHistory();
  const { from, to } = useTimeRange();
  const [historyOpen, setHistoryOpen] = useState(false);

  const busy =
    status === "thinking" || status === "querying" || status === "streaming";

  const investigate = (problem: string) => {
    void start(problem, { from, to });
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <span className={styles.title}>Case file</span>
        <div className={styles.spacer} />
        <IconButton label="Case history" onClick={() => setHistoryOpen(true)}>
          <History size={16} />
        </IconButton>
        <IconButton label="New case" onClick={clearSession}>
          <FilePlus2 size={16} />
        </IconButton>
        <IconButton label="Agent settings" onClick={openSettings}>
          <Settings size={16} />
        </IconButton>
      </header>

      <div className={styles.transcriptWrap}>
        <CaseTranscript turns={turns} status={status} onExample={investigate} />
      </div>

      <Composer
        busy={busy}
        disabled={!config.openaiApiKey}
        onSubmit={investigate}
        onStop={stop}
      />

      <SettingsModal
        open={isOpen}
        onClose={closeSettings}
        config={config}
        onSave={saveConfig}
      />
      <HistoryPanel
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
        sessions={sessions}
        onSelect={loadSession}
        onDelete={deleteSession}
      />
    </div>
  );
};

export default AgentPage;

import React, { useEffect, useRef } from "react";
import type { CaseTurn, InvestigationStatus } from "../../types/agent";
import { Alert, Chip, Spinner } from "../../ui";
import { SimpleBarLine } from "../charts/SimpleBarLine";
import { MarkdownText } from "./MarkdownText";
import { ExhibitCard } from "./ExhibitCard";
import styles from "./CaseTranscript.module.css";

const EXAMPLES = [
  "Why are pods crash-looping in kube-system?",
  "What caused the spike in warning events?",
  "Which nodes are having disk pressure issues?",
];

interface Props {
  turns: CaseTurn[];
  status: InvestigationStatus;
  onExample: (text: string) => void;
}

const STATUS_LINES: Partial<Record<InvestigationStatus, string>> = {
  thinking: "forming a hypothesis…",
  querying: "gathering evidence…",
  streaming: "writing findings…",
};

export const CaseTranscript: React.FC<Props> = ({
  turns,
  status,
  onExample,
}) => {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [turns, status]);

  if (turns.length === 0) {
    return (
      <div className={styles.empty}>
        <div className={styles.emptyTitle}>Open a case</div>
        <div className={styles.emptyHint}>
          Describe a problem and the agent investigates it against the event
          archive — every query it runs is filed as a numbered exhibit.
        </div>
        <div className={styles.examples}>
          {EXAMPLES.map((example) => (
            <Chip key={example} onClick={() => onExample(example)}>
              {example}
            </Chip>
          ))}
        </div>
      </div>
    );
  }

  const statusLine = STATUS_LINES[status];

  return (
    <div className={styles.transcript}>
      {turns.map((turn) =>
        turn.role === "user" ? (
          <div key={turn.id} className={styles.turn}>
            <div className={styles.eyebrow}>Statement</div>
            <div className={styles.statement}>{turn.text}</div>
          </div>
        ) : (
          <div key={turn.id} className={styles.turn}>
            {turn.exhibits.length > 0 && (
              <div className={styles.eyebrow}>Evidence</div>
            )}
            {turn.exhibits.map((exhibit) => (
              <ExhibitCard key={exhibit.id} exhibit={exhibit} />
            ))}
            {turn.charts.map((chart, i) => (
              <div key={i}>
                <div className={styles.eyebrow}>{chart.title}</div>
                <SimpleBarLine
                  type={chart.type}
                  content={chart.content}
                  height={240}
                />
              </div>
            ))}
            {turn.text && (
              <>
                <div className={styles.eyebrow}>Findings</div>
                <MarkdownText>{turn.text}</MarkdownText>
              </>
            )}
            {turn.error && <Alert tone="error">{turn.error}</Alert>}
          </div>
        ),
      )}

      {statusLine && (
        <div className={styles.status}>
          <Spinner size={12} />
          {statusLine}
        </div>
      )}
      <div ref={bottomRef} />
    </div>
  );
};

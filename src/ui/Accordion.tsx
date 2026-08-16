import React from "react";
import { ChevronRight } from "lucide-react";
import styles from "./Accordion.module.css";
import { cx } from "./cx";

export interface AccordionProps {
  summary: React.ReactNode;
  defaultOpen?: boolean;
  children: React.ReactNode;
  className?: string;
}

export const Accordion: React.FC<AccordionProps> = ({
  summary,
  defaultOpen = false,
  children,
  className,
}) => (
  <details className={cx(styles.accordion, className)} open={defaultOpen}>
    <summary className={styles.summary}>
      <ChevronRight size={14} className={styles.chevron} />
      {summary}
    </summary>
    <div className={styles.body}>{children}</div>
  </details>
);

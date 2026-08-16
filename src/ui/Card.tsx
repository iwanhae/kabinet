import React from "react";
import styles from "./Card.module.css";
import { cx } from "./cx";

export interface CardProps {
  title?: React.ReactNode;
  actions?: React.ReactNode;
  /** Removes body padding (for tables/charts that manage their own). */
  flush?: boolean;
  children: React.ReactNode;
  className?: string;
}

export const Card: React.FC<CardProps> = ({
  title,
  actions,
  flush = false,
  children,
  className,
}) => (
  <section className={cx(styles.card, flush && styles.flush, className)}>
    {(title || actions) && (
      <header className={styles.header}>
        {title && <h2 className={styles.title}>{title}</h2>}
        {actions}
      </header>
    )}
    <div className={styles.body}>{children}</div>
  </section>
);

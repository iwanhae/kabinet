import React from "react";
import { X } from "lucide-react";
import styles from "./Chip.module.css";
import { cx } from "./cx";

export type ChipTone = "neutral" | "steel" | "signal" | "alert";

export interface ChipProps {
  children: React.ReactNode;
  tone?: ChipTone;
  onClick?: (e: React.MouseEvent<HTMLButtonElement>) => void;
  onRemove?: () => void;
  title?: string;
  className?: string;
}

export const Chip: React.FC<ChipProps> = ({
  children,
  tone = "neutral",
  onClick,
  onRemove,
  title,
  className,
}) => {
  const cls = cx(
    styles.chip,
    tone !== "neutral" && styles[tone],
    onClick && styles.clickable,
    className,
  );

  const body = (
    <>
      <span className={styles.label}>{children}</span>
      {onRemove && (
        <button
          type="button"
          className={styles.remove}
          aria-label="Remove"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
        >
          <X size={12} />
        </button>
      )}
    </>
  );

  if (onClick) {
    return (
      <button type="button" className={cls} onClick={onClick} title={title}>
        {body}
      </button>
    );
  }
  return (
    <span className={cls} title={title}>
      {body}
    </span>
  );
};

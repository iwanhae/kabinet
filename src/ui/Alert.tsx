import React from "react";
import { AlertTriangle, Info, OctagonX } from "lucide-react";
import styles from "./Alert.module.css";
import { cx } from "./cx";

export interface AlertProps {
  tone?: "info" | "warning" | "error";
  children: React.ReactNode;
  className?: string;
}

const icons = {
  info: Info,
  warning: AlertTriangle,
  error: OctagonX,
} as const;

export const Alert: React.FC<AlertProps> = ({
  tone = "info",
  children,
  className,
}) => {
  const Icon = icons[tone];
  return (
    <div
      role={tone === "error" ? "alert" : "status"}
      className={cx(styles.alert, tone !== "info" && styles[tone], className)}
    >
      <Icon size={16} style={{ flexShrink: 0, marginTop: 2 }} />
      <div>{children}</div>
    </div>
  );
};

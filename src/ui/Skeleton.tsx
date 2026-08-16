import React from "react";
import styles from "./Skeleton.module.css";
import { cx } from "./cx";

export interface SkeletonProps {
  width?: number | string;
  height?: number | string;
  className?: string;
}

export const Skeleton: React.FC<SkeletonProps> = ({
  width = "100%",
  height = 16,
  className,
}) => (
  <span
    className={cx(styles.skeleton, className)}
    style={{ width, height }}
    aria-hidden="true"
  />
);

import React from "react";
import styles from "./Spinner.module.css";

export const Spinner: React.FC<{ size?: number }> = ({ size = 20 }) => (
  <span
    className={styles.spinner}
    style={{ width: size, height: size }}
    role="progressbar"
    aria-label="Loading"
  />
);

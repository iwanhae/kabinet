import React from "react";
import ReactMarkdown from "react-markdown";
import styles from "./MarkdownText.module.css";

export const MarkdownText: React.FC<{ children: string }> = ({ children }) => (
  <div className={styles.markdown}>
    <ReactMarkdown>{children}</ReactMarkdown>
  </div>
);

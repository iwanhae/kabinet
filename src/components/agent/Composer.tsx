import React, { useState } from "react";
import { Send, Square } from "lucide-react";
import { Button, TextArea } from "../../ui";
import styles from "./Composer.module.css";

interface Props {
  busy: boolean;
  disabled: boolean;
  onSubmit: (text: string) => void;
  onStop: () => void;
}

export const Composer: React.FC<Props> = ({
  busy,
  disabled,
  onSubmit,
  onStop,
}) => {
  const [text, setText] = useState("");

  const submit = () => {
    const trimmed = text.trim();
    if (!trimmed || busy || disabled) return;
    onSubmit(trimmed);
    setText("");
  };

  return (
    <div className={styles.composer}>
      <TextArea
        className={styles.input}
        rows={2}
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" && !e.shiftKey) {
            e.preventDefault();
            submit();
          }
        }}
        placeholder={
          disabled
            ? "Configure the OpenAI API key in settings first"
            : "Describe the problem to investigate… (Enter to send)"
        }
        disabled={disabled}
      />
      {busy ? (
        <Button variant="outline" onClick={onStop}>
          <Square size={14} />
          Stop
        </Button>
      ) : (
        <Button
          variant="solid"
          onClick={submit}
          disabled={disabled || !text.trim()}
        >
          <Send size={14} />
          Investigate
        </Button>
      )}
    </div>
  );
};

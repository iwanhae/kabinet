import React, { useEffect, useState } from "react";
import type { InvestigationConfig } from "../../types/agent";
import { Button, Modal, TextInput } from "../../ui";

interface Props {
  open: boolean;
  onClose: () => void;
  config: InvestigationConfig;
  onSave: (config: InvestigationConfig) => void;
}

export const SettingsModal: React.FC<Props> = ({
  open,
  onClose,
  config,
  onSave,
}) => {
  const [draft, setDraft] = useState(config);

  useEffect(() => {
    if (open) setDraft(config);
  }, [open, config]);

  const set = (patch: Partial<InvestigationConfig>) =>
    setDraft((d) => ({ ...d, ...patch }));

  return (
    <Modal open={open} onClose={onClose} title="Agent settings">
      <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
        <TextInput
          label="OpenAI API key"
          type="password"
          mono
          value={draft.openaiApiKey}
          onChange={(e) => set({ openaiApiKey: e.target.value })}
          placeholder="sk-…"
        />
        <TextInput
          label="API base URL"
          mono
          value={draft.openaiApiBase}
          onChange={(e) => set({ openaiApiBase: e.target.value })}
          placeholder="https://api.openai.com/v1"
        />
        <TextInput
          label="Model"
          mono
          value={draft.openaiModel ?? ""}
          onChange={(e) => set({ openaiModel: e.target.value })}
          placeholder="gpt-4o"
        />
        <TextInput
          label="Kabinet query endpoint"
          mono
          value={draft.kubeApiUrl}
          onChange={(e) => set({ kubeApiUrl: e.target.value })}
          placeholder="/query"
        />
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="solid"
            onClick={() => onSave(draft)}
            disabled={!draft.openaiApiKey}
          >
            Save settings
          </Button>
        </div>
      </div>
    </Modal>
  );
};

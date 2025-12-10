import React, { useState, useEffect } from "react";
import type { InvestigationConfig } from "../../types/agent";
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  Typography,
  Box,
  Alert,
} from "@mui/material";

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  config: InvestigationConfig;
  onSave: (config: InvestigationConfig) => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  config,
  onSave,
}) => {
  const [apiKey, setApiKey] = useState(config.openaiApiKey || "");
  const [apiBase, setApiBase] = useState(config.openaiApiBase || "");
  const [model, setModel] = useState(config.openaiModel || "gpt-4o");

  useEffect(() => {
    setApiKey(config.openaiApiKey || "");
    setApiBase(config.openaiApiBase || "");
    setModel(config.openaiModel || "gpt-5.1-mini");
  }, [config, isOpen]);

  const handleSave = () => {
    onSave({
      ...config,
      openaiApiKey: apiKey,
      openaiApiBase: apiBase,
      openaiModel: model,
    });
    onClose();
  };

  return (
    <Dialog open={isOpen} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Settings</DialogTitle>
      <DialogContent>
        <Box sx={{ mt: 1 }}>
          <Typography variant="body2" color="text.secondary" paragraph>
            Configure the AI agent settings.
          </Typography>

          <TextField
            autoFocus
            margin="dense"
            label="OpenAI API Key"
            type="password"
            fullWidth
            variant="outlined"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            helperText="Your key is stored locally in your browser."
          />

          <TextField
            margin="dense"
            label="OpenAI API Base URL"
            type="text"
            fullWidth
            variant="outlined"
            value={apiBase}
            onChange={(e) => setApiBase(e.target.value)}
            placeholder="https://api.openai.com/v1"
            helperText="Optional. Use for custom endpoints or proxies."
          />

          <TextField
            margin="dense"
            label="Model"
            type="text"
            fullWidth
            variant="outlined"
            value={model}
            onChange={(e) => setModel(e.target.value)}
            placeholder="gpt-4o"
            helperText="OpenAI model to use (e.g., gpt-4o, gpt-4-turbo, gpt-3.5-turbo)"
          />

          {!apiKey && (
            <Alert severity="warning" sx={{ mt: 2 }}>
              An OpenAI API Key is required for the agent to function.
            </Alert>
          )}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={!apiKey}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
};

import React, { useState, useEffect } from 'react';
import type { InvestigationConfig } from '../../types/agent';
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    Button,
    Typography,
    Box,
    Alert
} from '@mui/material';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
    config: InvestigationConfig;
    onSave: (config: InvestigationConfig) => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose, config, onSave }) => {
    const [apiKey, setApiKey] = useState(config.openaiApiKey || '');
    const [apiBase, setApiBase] = useState(config.openaiApiBase || '');

    useEffect(() => {
        setApiKey(config.openaiApiKey || '');
        setApiBase(config.openaiApiBase || '');
    }, [config, isOpen]);

    const handleSave = () => {
        onSave({ ...config, openaiApiKey: apiKey, openaiApiBase: apiBase });
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

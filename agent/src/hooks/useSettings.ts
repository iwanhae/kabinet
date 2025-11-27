import { useState, useEffect } from 'react';
import type { InvestigationConfig } from '../lib/types';

const DEFAULT_CONFIG: InvestigationConfig = {
    openaiApiKey: '',
    openaiApiBase: 'https://api.openai.com/v1',
    kubeApiUrl: 'http://127.0.0.1:8080/query'
};

export const useSettings = () => {
    const [config, setConfig] = useState<InvestigationConfig>(DEFAULT_CONFIG);
    const [isOpen, setIsOpen] = useState(false);

    useEffect(() => {
        const stored = localStorage.getItem('agent_config');
        if (stored) {
            try {
                setConfig({ ...DEFAULT_CONFIG, ...JSON.parse(stored) });
            } catch (e) {
                console.error("Failed to parse settings", e);
            }
        } else {
            setIsOpen(true); // Open settings on first load if no config
        }
    }, []);

    const saveConfig = (newConfig: InvestigationConfig) => {
        setConfig(newConfig);
        localStorage.setItem('agent_config', JSON.stringify(newConfig));
        setIsOpen(false);
    };

    return {
        config,
        saveConfig,
        isOpen,
        openSettings: () => setIsOpen(true),
        closeSettings: () => setIsOpen(false)
    };
};

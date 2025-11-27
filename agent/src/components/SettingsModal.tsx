import React, { useState, useEffect } from 'react';
import type { InvestigationConfig } from '../lib/types';
import { X, Settings, Save } from 'lucide-react';

interface SettingsModalProps {
    isOpen: boolean;
    onClose: () => void;
    config: InvestigationConfig;
    onSave: (config: InvestigationConfig) => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose, config, onSave }) => {
    const [formData, setFormData] = useState(config);

    useEffect(() => {
        setFormData(config);
    }, [config, isOpen]);

    if (!isOpen) return null;

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        onSave(formData);
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm animate-fade-in p-4">
            <div className="w-full max-w-lg bg-bg-panel border border-gray-800 rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
                <div className="flex items-center justify-between p-6 border-b border-gray-800">
                    <h2 className="text-xl font-semibold flex items-center gap-2 text-text-primary">
                        <Settings className="w-5 h-5 text-accent-primary" />
                        Configuration
                    </h2>
                    <button onClick={onClose} className="text-text-tertiary hover:text-text-primary transition-colors p-1 hover:bg-gray-800 rounded-full">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                <form onSubmit={handleSubmit} className="p-6 space-y-6 overflow-y-auto">
                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-text-secondary">
                            OpenAI API Key
                        </label>
                        <input
                            type="password"
                            value={formData.openaiApiKey}
                            onChange={e => setFormData({ ...formData, openaiApiKey: e.target.value })}
                            className="w-full bg-bg-input border border-gray-700 rounded-lg px-4 py-3 text-text-primary focus:outline-none focus:ring-2 focus:ring-accent-primary/50 focus:border-accent-primary transition-all"
                            placeholder="sk-..."
                            required
                        />
                        <p className="text-xs text-text-tertiary">Your key is stored locally in your browser.</p>
                    </div>

                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-text-secondary">
                            OpenAI Base URL
                        </label>
                        <input
                            type="text"
                            value={formData.openaiApiBase}
                            onChange={e => setFormData({ ...formData, openaiApiBase: e.target.value })}
                            className="w-full bg-bg-input border border-gray-700 rounded-lg px-4 py-3 text-text-primary focus:outline-none focus:ring-2 focus:ring-accent-primary/50 focus:border-accent-primary transition-all"
                            placeholder="https://api.openai.com/v1"
                        />
                    </div>

                    <div className="space-y-2">
                        <label className="block text-sm font-medium text-text-secondary">
                            Kabinet URL
                        </label>
                        <input
                            type="text"
                            value={formData.kubeApiUrl}
                            onChange={e => setFormData({ ...formData, kubeApiUrl: e.target.value })}
                            className="w-full bg-bg-input border border-gray-700 rounded-lg px-4 py-3 text-text-primary focus:outline-none focus:ring-2 focus:ring-accent-primary/50 focus:border-accent-primary transition-all"
                            placeholder="http://localhost:8080/query"
                        />
                    </div>
                </form>

                <div className="p-6 border-t border-gray-800 flex justify-end gap-3 bg-bg-panel">
                    <button
                        type="button"
                        onClick={onClose}
                        className="px-5 py-2.5 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-gray-800 rounded-lg transition-colors"
                    >
                        Cancel
                    </button>
                    <button
                        onClick={handleSubmit}
                        type="submit"
                        className="px-5 py-2.5 text-sm font-medium bg-accent-primary text-bg-app rounded-lg hover:opacity-90 transition-opacity flex items-center gap-2"
                    >
                        <Save className="w-4 h-4" />
                        Save Configuration
                    </button>
                </div>
            </div>
        </div>
    );
};

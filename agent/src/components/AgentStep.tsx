import React from 'react';
import type { InvestigationStatus, AgentPlan } from '../lib/types';
import { Brain, Search, Lightbulb, Loader2, CheckCircle2, Database, Clock } from 'lucide-react';

interface AgentStepProps {
    status: InvestigationStatus;
    thought: string;
    hypothesis: string;
    query?: AgentPlan['query'];
}

export const AgentStep: React.FC<AgentStepProps> = ({ status, thought, hypothesis, query }) => {
    if (status === 'idle' || status === 'error') return null;

    const isComplete = status === 'complete';

    return (
        <div className="flex flex-col gap-4 p-5 my-6 rounded-2xl bg-bg-panel border border-gray-800 shadow-lg animate-fade-in relative overflow-hidden">
            {/* Decorative gradient background */}
            <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-blue-500 via-purple-500 to-blue-500 animate-gradient-x"></div>

            {/* Status Header */}
            <div className="flex items-center gap-3 text-sm font-medium">
                <div className={`p-2 rounded-full ${isComplete ? 'bg-green-500/10 text-green-400' : 'bg-blue-500/10 text-accent-primary'}`}>
                    {status === 'planning' && <Brain className="w-4 h-4 animate-pulse" />}
                    {status === 'querying' && <Search className="w-4 h-4 animate-bounce" />}
                    {status === 'analyzing' && <Loader2 className="w-4 h-4 animate-spin" />}
                    {status === 'complete' && <CheckCircle2 className="w-4 h-4" />}
                </div>

                <span className="capitalize text-text-primary">
                    {isComplete ? 'Investigation Complete' : `${status}...`}
                </span>
            </div>

            {/* Thought Process */}
            {thought && !isComplete && (
                <div className="text-sm text-text-secondary pl-11 relative">
                    <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-800"></div>
                    <span className="italic">"{thought}"</span>
                </div>
            )}

            {/* Current Hypothesis */}
            {hypothesis && !isComplete && (
                <div className="ml-11 p-4 rounded-xl bg-bg-app border border-gray-800/50 flex gap-3">
                    <Lightbulb className="w-5 h-5 text-yellow-500 shrink-0 mt-0.5" />
                    <div>
                        <div className="text-xs font-bold text-text-tertiary uppercase tracking-wider mb-1">
                            Working Hypothesis
                        </div>
                        <div className="text-sm text-text-primary leading-relaxed">
                            {hypothesis}
                        </div>
                    </div>
                </div>
            )}

            {/* Active Query */}
            {query && status === 'querying' && (
                <div className="ml-11 mt-2">
                    <div className="rounded-xl bg-black/30 border border-gray-800 overflow-hidden">
                        <div className="flex items-center gap-2 px-4 py-2 bg-gray-800/30 border-b border-gray-800">
                            <Database className="w-3.5 h-3.5 text-accent-primary" />
                            <span className="text-xs font-medium text-text-secondary">Executing SQL Query</span>
                        </div>
                        <div className="p-4 space-y-3">
                            <div className="font-mono text-sm text-green-400 break-all whitespace-pre-wrap">
                                {query.sql}
                            </div>
                            <div className="flex items-center gap-4 text-xs text-text-tertiary font-mono">
                                <div className="flex items-center gap-1.5">
                                    <Clock className="w-3 h-3" />
                                    <span>Start: {new Date(query.start).toLocaleString()}</span>
                                </div>
                                <div className="flex items-center gap-1.5">
                                    <Clock className="w-3 h-3" />
                                    <span>End: {new Date(query.end).toLocaleString()}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};

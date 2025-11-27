import React from 'react';
import type { Message, AgentPlan } from '../lib/types';
import { DataVisualizer } from './DataVisualizer';
import { User, Bot, Terminal, Brain, Lightbulb, Database, CheckCircle2, ChevronDown, ChevronRight } from 'lucide-react';

interface MessageBubbleProps {
    message: Message;
}

export const MessageBubble: React.FC<MessageBubbleProps> = ({ message }) => {
    const isUser = message.role === 'user';
    const isSystem = message.role === 'system';

    const [isExpanded, setIsExpanded] = React.useState(false);
    const isQueryResult = message.content.startsWith('Query executed. Result:');

    if (isSystem) {
        if (isQueryResult) {
            const jsonContent = message.content.replace('Query executed. Result:\n', '');
            return (
                <div className="flex flex-col gap-2 p-4 rounded-xl bg-bg-panel/50 border border-gray-800/50 font-mono text-sm text-text-secondary animate-fade-in overflow-hidden">
                    <div
                        className="flex items-center gap-2 cursor-pointer hover:text-text-primary transition-colors"
                        onClick={() => setIsExpanded(!isExpanded)}
                    >
                        <Database className="w-4 h-4 text-accent-primary shrink-0" />
                        <span className="font-medium">Query Result</span>
                        {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                    </div>

                    {isExpanded && (
                        <div className="mt-2 overflow-x-auto bg-black/30 p-3 rounded-lg border border-gray-800/50">
                            <pre className="text-xs text-green-400 whitespace-pre-wrap break-all">
                                {jsonContent}
                            </pre>
                        </div>
                    )}
                </div>
            );
        }

        return (
            <div className="flex gap-4 p-4 rounded-xl bg-bg-panel/50 border border-gray-800/50 font-mono text-sm text-text-secondary animate-fade-in">
                <Terminal className="w-5 h-5 text-text-tertiary shrink-0 mt-0.5" />
                <div className="whitespace-pre-wrap leading-relaxed opacity-90">
                    {message.content}
                </div>
            </div>
        );
    }

    let content = message.content;
    let parsedPlan: AgentPlan | null = null;

    if (!isUser) {
        try {
            parsedPlan = JSON.parse(message.content);
        } catch (e) {
            // Not JSON, render as text
        }
    }

    return (
        <div className={`flex gap-4 ${isUser ? 'flex-row-reverse' : ''} animate-fade-in group`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 shadow-lg ${isUser
                ? 'bg-gradient-to-br from-blue-500 to-blue-600'
                : 'bg-gradient-to-br from-purple-500 to-purple-600'
                }`}>
                {isUser ? <User className="w-5 h-5 text-white" /> : <Bot className="w-5 h-5 text-white" />}
            </div>

            <div className={`max-w-[85%] rounded-2xl px-6 py-4 shadow-sm ${isUser
                ? 'bg-bg-panel text-text-primary rounded-tr-sm'
                : 'bg-bg-panel/50 text-text-primary rounded-tl-sm border border-gray-800/50'
                }`}>
                {parsedPlan ? (
                    <div className="space-y-4">
                        {parsedPlan.thought && (
                            <div className="flex gap-3 text-text-secondary">
                                <Brain className="w-4 h-4 mt-1 shrink-0" />
                                <div className="italic text-sm">"{parsedPlan.thought}"</div>
                            </div>
                        )}

                        {parsedPlan.hypothesis && (
                            <div className="flex gap-3 bg-bg-app/50 p-3 rounded-lg border border-gray-800/50">
                                <Lightbulb className="w-4 h-4 text-yellow-500 mt-1 shrink-0" />
                                <div>
                                    <div className="text-xs font-bold text-text-tertiary uppercase tracking-wider mb-1">Hypothesis</div>
                                    <div className="text-sm">{parsedPlan.hypothesis}</div>
                                </div>
                            </div>
                        )}

                        {parsedPlan.query && (
                            <div className="flex gap-3">
                                <Database className="w-4 h-4 text-accent-primary mt-1 shrink-0" />
                                <div className="flex-1 min-w-0">
                                    <div className="text-xs font-bold text-text-tertiary uppercase tracking-wider mb-1">Executing Query</div>
                                    <div className="bg-black/30 rounded-lg p-3 font-mono text-xs text-green-400 overflow-x-auto">
                                        {parsedPlan.query.sql}
                                    </div>
                                </div>
                            </div>
                        )}

                        {parsedPlan.final_analysis && (
                            <div className="flex gap-3 border-t border-gray-800/50 pt-3 mt-2">
                                <CheckCircle2 className="w-4 h-4 text-green-500 mt-1 shrink-0" />
                                <div>
                                    <div className="text-xs font-bold text-text-tertiary uppercase tracking-wider mb-1">Conclusion</div>
                                    <div className="text-base leading-relaxed">{parsedPlan.final_analysis}</div>
                                </div>
                            </div>
                        )}

                        {parsedPlan.data && (
                            <DataVisualizer data={parsedPlan.data} />
                        )}
                    </div>
                ) : (
                    <div className="whitespace-pre-wrap leading-relaxed text-base">
                        {content}
                    </div>
                )}
            </div>
        </div>
    );
};

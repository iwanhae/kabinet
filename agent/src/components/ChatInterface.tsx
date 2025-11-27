import React, { useRef, useEffect, useState } from 'react';
import type { Message, InvestigationStatus, AgentPlan } from '../lib/types';
import { MessageBubble } from './MessageBubble';
import { AgentStep } from './AgentStep';
import { Send, StopCircle, Sparkles } from 'lucide-react';

interface ChatInterfaceProps {
    messages: Message[];
    status: InvestigationStatus;
    currentThought: string;
    currentHypothesis: string;
    currentQuery?: AgentPlan['query'];
    onStartInvestigation: (problem: string) => void;
    onStop: () => void;
}

export const ChatInterface: React.FC<ChatInterfaceProps> = ({
    messages,
    status,
    currentThought,
    currentHypothesis,
    currentQuery,
    onStartInvestigation,
    onStop
}) => {
    const [input, setInput] = useState('');
    const bottomRef = useRef<HTMLDivElement>(null);
    const isBusy = status !== 'idle' && status !== 'complete' && status !== 'error';

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages, currentThought, status]);

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!input.trim()) return;
        onStartInvestigation(input);
        setInput('');
    };

    return (
        <div className="flex flex-col h-full w-full relative">
            {/* Messages Area */}
            <div className="flex-1 overflow-y-auto p-10 pb-32">
                <div className='max-w-5xl mx-auto flex-1 space-y-8'>
                    {messages.length === 0 && (
                        <div className="h-full flex flex-col items-center justify-center text-center text-text-tertiary opacity-70 animate-fade-in">
                            <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 flex items-center justify-center mb-6 shadow-lg shadow-blue-500/5">
                                <Sparkles className="w-10 h-10 text-accent-primary" />
                            </div>
                            <h1 className="text-4xl font-medium bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent mb-3">
                                Hello, Human
                            </h1>
                            <p className="text-xl text-text-secondary max-w-md">
                                I can help you troubleshoot Kubernetes cluster events. What seems to be the problem?
                            </p>
                        </div>
                    )}

                    {messages.filter(msg =>
                        msg.role !== 'system' ||
                        msg.content.startsWith('Query executed. Result:') ||
                        msg.content.startsWith('Error:') ||
                        msg.content.startsWith('AI did not provide') ||
                        msg.content.startsWith('Maximum turns reached')
                    ).map((msg, idx) => (
                        <MessageBubble key={idx} message={msg} />
                    ))}

                    <AgentStep
                        status={status}
                        thought={currentThought}
                        hypothesis={currentHypothesis}
                        query={currentQuery}
                    />
                </div>
                <div ref={bottomRef} />
            </div>

            {/* Input Area */}
            <div className="absolute bottom-0 left-0 right-0 p-6 bg-gradient-to-t from-bg-app via-bg-app to-transparent">
                <div className="max-w-3xl mx-auto">
                    <form onSubmit={handleSubmit} className="relative group">
                        <div className="absolute -inset-0.5 bg-gradient-to-r from-blue-500 to-purple-600 rounded-2xl opacity-20 group-hover:opacity-40 transition duration-500 blur"></div>
                        <div className="relative flex items-center bg-bg-input rounded-2xl border border-gray-700/50 shadow-xl">
                            <input
                                type="text"
                                value={input}
                                onChange={(e) => setInput(e.target.value)}
                                placeholder={isBusy ? "Investigation in progress..." : "Describe the problem (e.g., 'Pods are failing on node-1')..."}
                                disabled={isBusy}
                                className="w-full bg-transparent border-none py-4 pl-6 pr-14 text-text-primary placeholder-text-tertiary focus:ring-0 focus:outline-none disabled:opacity-50 text-lg"
                            />

                            <div className="absolute right-3">
                                {isBusy ? (
                                    <button
                                        type="button"
                                        onClick={onStop}
                                        className="p-2 text-red-400 hover:bg-red-400/10 rounded-xl transition-colors"
                                        title="Stop Investigation"
                                    >
                                        <StopCircle className="w-6 h-6" />
                                    </button>
                                ) : (
                                    <button
                                        type="submit"
                                        disabled={!input.trim()}
                                        className="p-2 text-accent-primary hover:bg-accent-primary/10 rounded-xl disabled:opacity-30 transition-all duration-200"
                                    >
                                        <Send className="w-6 h-6" />
                                    </button>
                                )}
                            </div>
                        </div>
                        <div className="text-center mt-3 text-xs text-text-tertiary">
                            Kabinet can make mistakes. Consider checking important information.
                        </div>
                    </form>
                </div>
            </div>
        </div>
    );
};

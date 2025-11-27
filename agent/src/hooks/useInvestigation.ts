import { useState, useCallback, useRef } from 'react';
import type { Message, InvestigationConfig, InvestigationStatus, AgentPlan } from '../lib/types';
import { createOpenAIClient } from '../lib/openai';
import { executeKubeQuery } from '../lib/kube';
import { SYSTEM_PROMPT } from '../lib/prompts';


export const useInvestigation = (config: InvestigationConfig) => {
    const [messages, setMessages] = useState<Message[]>([]);
    const [status, setStatus] = useState<InvestigationStatus>('idle');
    const [currentHypothesis, setCurrentHypothesis] = useState<string>('');
    const [currentThought, setCurrentThought] = useState<string>('');
    const [currentQuery, setCurrentQuery] = useState<AgentPlan['query'] | undefined>(undefined);

    // Refs to access latest state in async loop without dependency issues
    const messagesRef = useRef<Message[]>([]);
    const stopRef = useRef<boolean>(false);

    const addMessage = (msg: Message) => {
        setMessages(prev => {
            const next = [...prev, msg];
            messagesRef.current = next;
            return next;
        });
    };

    const stop = useCallback(() => {
        stopRef.current = true;
        setStatus('idle'); // Or 'cancelled'
    }, []);

    const start = useCallback(async (userProblem: string) => {
        if (!config.openaiApiKey) {
            alert("Please configure OpenAI API Key first.");
            return;
        }

        // Reset state for new turn, but keep history if it exists
        stopRef.current = false;
        setStatus('planning');
        setCurrentHypothesis('Initializing investigation...');
        setCurrentThought('Analyzing user request...');
        setCurrentQuery(undefined);

        let currentMessages = messagesRef.current;

        if (currentMessages.length === 0) {
            // New session
            currentMessages = [
                { role: 'system', content: SYSTEM_PROMPT },
                { role: 'user', content: userProblem }
            ];
        } else {
            // Continuing session
            currentMessages = [
                ...currentMessages,
                { role: 'user', content: userProblem }
            ];
        }

        // Update state with new messages
        setMessages(currentMessages);
        messagesRef.current = currentMessages;

        const client = createOpenAIClient(config);
        let turn = 0;
        const maxTurns = 15;

        try {
            while (turn < maxTurns && !stopRef.current) {
                turn++;
                setStatus('planning');

                // Prepare context for AI
                const currentTime = new Date().toISOString();
                const contextMessages = [
                    ...messagesRef.current,
                    { role: 'system', content: `The current UTC time is ${currentTime}. Use this to construct your query's time range.` }
                ] as any[]; // Type cast for OpenAI SDK compatibility

                // 1. Get AI Plan
                const completion = await client.chat.completions.create({
                    model: "gpt-4.1",
                    messages: contextMessages,
                    response_format: { type: "json_object" }
                });

                const content = completion.choices[0].message.content;
                if (!content) throw new Error("Empty response from AI");

                const plan: AgentPlan = JSON.parse(content);

                // Update UI with AI's thought process
                addMessage({ role: 'assistant', content: content }); // Store raw JSON for history
                if (plan.thought) setCurrentThought(plan.thought);
                if (plan.hypothesis) setCurrentHypothesis(plan.hypothesis);

                // 2. Check for conclusion
                if (plan.final_analysis) {
                    setStatus('complete');
                    return;
                }

                // 3. Execute Query
                if (plan.query) {
                    setStatus('querying');
                    setCurrentQuery(plan.query);
                    const result = await executeKubeQuery(config, plan.query.sql, plan.query.start, plan.query.end);

                    setStatus('analyzing');
                    setCurrentQuery(undefined);
                    const summary = JSON.stringify(result, null, 2);

                    addMessage({
                        role: 'system',
                        content: `Query executed. Result:\n${summary}`
                    });
                } else {
                    // Fallback if no query and no conclusion (shouldn't happen with good prompt)
                    addMessage({ role: 'system', content: "AI did not provide a query or final analysis. Ending." });
                    break;
                }
            }

            if (turn >= maxTurns) {
                addMessage({ role: 'system', content: "Maximum turns reached." });
                setStatus('complete');
            }

        } catch (error: any) {
            console.error("Investigation error:", error);
            addMessage({ role: 'system', content: `Error: ${error.message}` });
            setStatus('error');
        }
    }, [config]);

    return {
        messages,
        status,
        currentHypothesis,
        currentThought,
        currentQuery,
        start,
        stop
    };
};

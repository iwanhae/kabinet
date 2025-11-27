import OpenAI from 'openai';
import type { InvestigationConfig } from './types';

export const createOpenAIClient = (config: InvestigationConfig) => {
    if (!config.openaiApiKey) {
        throw new Error("OpenAI API Key is missing");
    }

    return new OpenAI({
        apiKey: config.openaiApiKey,
        baseURL: config.openaiApiBase || undefined,
        dangerouslyAllowBrowser: true // Required for client-side usage
    });
};

import type { InvestigationConfig, QueryResult } from './types';

export const executeKubeQuery = async (
    config: InvestigationConfig,
    query: string,
    start: string,
    end: string
): Promise<QueryResult> => {
    console.log(`[KubeClient] Executing query: ${query} (${start} - ${end})`);

    try {
        const response = await fetch(config.kubeApiUrl, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                query,
                start,
                end,
            }),
        });

        if (!response.ok) {
            const text = await response.text();
            return { error: `API call failed: ${response.status} ${text}` };
        }

        const data = await response.json();
        return data;
    } catch (e: any) {
        return { error: `Network error: ${e.message}` };
    }
};

import { useState, useCallback, useRef, useEffect } from "react";
import type OpenAI from "openai";
import type {
  CaseSession,
  CaseTurn,
  Exhibit,
  InvestigationConfig,
  InvestigationStatus,
} from "../../types/agent";
import { useHistory } from "./useHistory";
import { createOpenAIClient } from "../../lib/agent/openai";
import { executeKubeQuery } from "../../lib/agent/kube";
import { SYSTEM_PROMPT, buildContextPrompt } from "../../lib/agent/prompts";

type ChatMessage = OpenAI.Chat.Completions.ChatCompletionMessageParam;

const MAX_TURNS = 15;
/** Rows kept for the exhibit preview UI. */
const PREVIEW_ROWS = 50;
/** Rows injected back into the model context (summarized, never the full dump). */
const CONTEXT_ROWS = 30;

const uuid = (): string =>
  typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : Math.random().toString(36).slice(2);

const TOOLS: OpenAI.Chat.Completions.ChatCompletionTool[] = [
  {
    type: "function",
    function: {
      name: "run_sql",
      description:
        "Run a DuckDB SQL query against the $events table of Kubernetes events. Returns the row count and up to 30 rows.",
      parameters: {
        type: "object",
        properties: {
          sql: {
            type: "string",
            description:
              "DuckDB SQL selecting FROM $events. Aggregate or LIMIT — never unbounded raw rows.",
          },
          start: {
            type: "string",
            description: "Scan range start, ISO 8601.",
          },
          end: { type: "string", description: "Scan range end, ISO 8601." },
        },
        required: ["sql", "start", "end"],
      },
    },
  },
  {
    type: "function",
    function: {
      name: "show_chart",
      description:
        "Render a bar or line chart to the user from result rows. Each row object needs one label key (label/name/date) and numeric series keys.",
      parameters: {
        type: "object",
        properties: {
          title: { type: "string" },
          chart_type: { type: "string", enum: ["bar", "line"] },
          content: { type: "array", items: { type: "object" } },
        },
        required: ["title", "chart_type", "content"],
      },
    },
  },
];

interface StreamedToolCall {
  id: string;
  name: string;
  arguments: string;
}

export const useInvestigation = (config: InvestigationConfig) => {
  const [turns, setTurns] = useState<CaseTurn[]>([]);
  const [status, setStatus] = useState<InvestigationStatus>("idle");
  const [sessionId, setSessionId] = useState<string>(() => uuid());

  const turnsRef = useRef<CaseTurn[]>([]);
  const chatRef = useRef<ChatMessage[]>([]);
  const stopRef = useRef(false);
  const exhibitSeqRef = useRef(0);

  const { saveSession } = useHistory();

  useEffect(() => {
    if (turns.length === 0) return;
    const userTurn = turns.find((t) => t.role === "user");
    const title = userTurn
      ? userTurn.text.slice(0, 60) + (userTurn.text.length > 60 ? "…" : "")
      : "New case";
    saveSession({ id: sessionId, timestamp: Date.now(), title, turns });
  }, [turns, sessionId, saveSession]);

  const commitTurns = useCallback(
    (updater: (prev: CaseTurn[]) => CaseTurn[]) => {
      setTurns((prev) => {
        const next = updater(prev);
        turnsRef.current = next;
        return next;
      });
    },
    [],
  );

  const patchTurn = useCallback(
    (turnId: string, patch: (t: CaseTurn) => CaseTurn) => {
      commitTurns((prev) => prev.map((t) => (t.id === turnId ? patch(t) : t)));
    },
    [commitTurns],
  );

  const stop = useCallback(() => {
    stopRef.current = true;
    setStatus("idle");
  }, []);

  const clearSession = useCallback(() => {
    stopRef.current = true;
    commitTurns(() => []);
    chatRef.current = [];
    exhibitSeqRef.current = 0;
    setStatus("idle");
    setSessionId(uuid());
  }, [commitTurns]);

  const loadSession = useCallback(
    (session: CaseSession) => {
      stopRef.current = true;
      setSessionId(session.id);
      commitTurns(() => session.turns);
      exhibitSeqRef.current = session.turns
        .flatMap((t) => t.exhibits)
        .reduce((m, e) => Math.max(m, e.seq), 0);
      // Rebuild the model conversation from the visible text (tool details are
      // summarized away — good enough for follow-up questions).
      chatRef.current = [
        { role: "system", content: SYSTEM_PROMPT },
        ...session.turns.map(
          (t): ChatMessage =>
            t.role === "user"
              ? { role: "user", content: t.text }
              : { role: "assistant", content: t.text || "(ran queries)" },
        ),
      ];
      setStatus("idle");
    },
    [commitTurns],
  );

  const runTool = useCallback(
    async (call: StreamedToolCall, turnId: string): Promise<string> => {
      let args: Record<string, unknown>;
      try {
        args = JSON.parse(call.arguments || "{}");
      } catch {
        return JSON.stringify({ error: "Invalid tool arguments (bad JSON)." });
      }

      if (call.name === "run_sql") {
        const sql = String(args.sql ?? "");
        const start = String(args.start ?? "");
        const end = String(args.end ?? "");
        const exhibit: Exhibit = {
          id: uuid(),
          seq: ++exhibitSeqRef.current,
          sql,
          start,
          end,
          status: "running",
        };
        patchTurn(turnId, (t) => ({
          ...t,
          exhibits: [...t.exhibits, exhibit],
        }));

        const result = await executeKubeQuery(config, sql, start, end);
        const rows = result.results ?? [];

        patchTurn(turnId, (t) => ({
          ...t,
          exhibits: t.exhibits.map((e) =>
            e.id === exhibit.id
              ? {
                  ...e,
                  status: result.error ? "error" : "done",
                  rowCount: rows.length,
                  durationMs: result.duration_ms,
                  rows: rows.slice(0, PREVIEW_ROWS),
                  error: result.error,
                }
              : e,
          ),
        }));

        if (result.error) return JSON.stringify({ error: result.error });
        return JSON.stringify({
          row_count: rows.length,
          columns: rows[0] ? Object.keys(rows[0]) : [],
          rows: rows.slice(0, CONTEXT_ROWS),
          truncated: rows.length > CONTEXT_ROWS,
          duration_ms: result.duration_ms,
        });
      }

      if (call.name === "show_chart") {
        const content = Array.isArray(args.content)
          ? (args.content as Record<string, unknown>[])
          : [];
        patchTurn(turnId, (t) => ({
          ...t,
          charts: [
            ...t.charts,
            {
              title: String(args.title ?? "Chart"),
              type: args.chart_type === "line" ? "line" : "bar",
              content,
            },
          ],
        }));
        return JSON.stringify({ ok: true, note: "Chart shown to the user." });
      }

      return JSON.stringify({ error: `Unknown tool: ${call.name}` });
    },
    [config, patchTurn],
  );

  const start = useCallback(
    async (problem: string, timeRange: { from: string; to: string }) => {
      if (!config.openaiApiKey) return;

      stopRef.current = false;
      if (chatRef.current.length === 0) {
        chatRef.current = [{ role: "system", content: SYSTEM_PROMPT }];
      }
      chatRef.current.push({ role: "user", content: problem });

      const userTurn: CaseTurn = {
        id: uuid(),
        role: "user",
        text: problem,
        exhibits: [],
        charts: [],
      };
      const assistantTurn: CaseTurn = {
        id: uuid(),
        role: "assistant",
        text: "",
        exhibits: [],
        charts: [],
      };
      commitTurns((prev) => [...prev, userTurn, assistantTurn]);

      const client = createOpenAIClient(config);
      let round = 0;

      try {
        while (round < MAX_TURNS && !stopRef.current) {
          round++;
          setStatus("thinking");

          const stream = await client.chat.completions.create({
            model: config.openaiModel || "gpt-4o",
            messages: [
              ...chatRef.current,
              {
                role: "system",
                content: buildContextPrompt(timeRange.from, timeRange.to),
              },
            ],
            tools: TOOLS,
            stream: true,
          });

          let content = "";
          const toolCalls = new Map<number, StreamedToolCall>();

          for await (const chunk of stream) {
            if (stopRef.current) {
              stream.controller.abort();
              break;
            }
            const delta = chunk.choices[0]?.delta;
            if (!delta) continue;
            if (delta.content) {
              content += delta.content;
              setStatus("streaming");
              patchTurn(assistantTurn.id, (t) => ({ ...t, text: content }));
            }
            for (const tc of delta.tool_calls ?? []) {
              let slot = toolCalls.get(tc.index);
              if (!slot) {
                slot = { id: "", name: "", arguments: "" };
                toolCalls.set(tc.index, slot);
              }
              if (tc.id) slot.id = tc.id;
              if (tc.function?.name) slot.name = tc.function.name;
              if (tc.function?.arguments)
                slot.arguments += tc.function.arguments;
            }
          }

          if (stopRef.current) break;

          const calls = [...toolCalls.values()];
          if (calls.length === 0) {
            chatRef.current.push({ role: "assistant", content });
            setStatus("complete");
            return;
          }

          chatRef.current.push({
            role: "assistant",
            content: content || null,
            tool_calls: calls.map((c) => ({
              id: c.id,
              type: "function",
              function: { name: c.name, arguments: c.arguments },
            })),
          });

          setStatus("querying");
          for (const call of calls) {
            if (stopRef.current) break;
            const payload = await runTool(call, assistantTurn.id);
            chatRef.current.push({
              role: "tool",
              tool_call_id: call.id,
              content: payload,
            });
          }
        }

        if (round >= MAX_TURNS) {
          patchTurn(assistantTurn.id, (t) => ({
            ...t,
            error: "Investigation stopped: maximum number of rounds reached.",
          }));
        }
        setStatus("complete");
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        patchTurn(assistantTurn.id, (t) => ({ ...t, error: message }));
        setStatus("error");
      }
    },
    [config, runTool, commitTurns, patchTurn],
  );

  return {
    turns,
    status,
    start,
    stop,
    clearSession,
    loadSession,
  };
};

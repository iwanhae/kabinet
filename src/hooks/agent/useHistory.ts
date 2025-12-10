import { useState, useEffect, useCallback } from "react";
import type { Message } from "../../types/agent";

export interface SavedSession {
  id: string;
  timestamp: number;
  title: string;
  messages: Message[];
}

const STORAGE_KEY = "kabinet_agent_sessions";

export const useHistory = () => {
  const [sessions, setSessions] = useState<SavedSession[]>([]);

  // Load sessions on mount
  useEffect(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        setSessions(JSON.parse(stored));
      }
    } catch (e) {
      console.error("Failed to load history:", e);
    }
  }, []);

  const saveSession = useCallback((session: SavedSession) => {
    setSessions((prev) => {
      const existingIndex = prev.findIndex((s) => s.id === session.id);
      let next;
      if (existingIndex >= 0) {
        next = [...prev];
        next[existingIndex] = session;
      } else {
        next = [session, ...prev];
      }
      // Sort by timestamp desc
      next.sort((a, b) => b.timestamp - a.timestamp);

      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  const deleteSession = useCallback((id: string) => {
    setSessions((prev) => {
      const next = prev.filter((s) => s.id !== id);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  const clearAll = useCallback(() => {
    setSessions([]);
    localStorage.removeItem(STORAGE_KEY);
  }, []);

  return {
    sessions,
    saveSession,
    deleteSession,
    clearAll,
  };
};

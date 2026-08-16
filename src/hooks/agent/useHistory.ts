import { useState, useEffect, useCallback } from "react";
import type { CaseSession } from "../../types/agent";

const STORAGE_KEY = "kabinet_agent_cases";

export const useHistory = () => {
  const [sessions, setSessions] = useState<CaseSession[]>([]);

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

  const saveSession = useCallback((session: CaseSession) => {
    setSessions((prev) => {
      const existingIndex = prev.findIndex((s) => s.id === session.id);
      let next;
      if (existingIndex >= 0) {
        next = [...prev];
        next[existingIndex] = session;
      } else {
        next = [session, ...prev];
      }
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

  return {
    sessions,
    saveSession,
    deleteSession,
  };
};

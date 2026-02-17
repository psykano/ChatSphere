import { useCallback, useState } from "react";
import type { Room } from "@/components/room-card";

const API_BASE = "/api";

interface UseJoinByCodeResult {
  joinByCode: (code: string) => Promise<Room | null>;
  loading: boolean;
  error: string | null;
  clearError: () => void;
}

export function useJoinByCode(): UseJoinByCodeResult {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearError = useCallback(() => setError(null), []);

  const joinByCode = useCallback(async (code: string): Promise<Room | null> => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/rooms/code/${code}`);
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.error ?? `Room not found (${res.status})`);
      }
      const room: Room = await res.json();
      return room;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to join room";
      setError(message);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return { joinByCode, loading, error, clearError };
}

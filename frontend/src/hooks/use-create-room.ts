import { useCallback, useState } from "react";
import type { Room } from "@/components/room-card";

const API_BASE = "/api";

interface CreateRoomInput {
  name: string;
  description: string;
  capacity: number;
  public: boolean;
}

interface UseCreateRoomResult {
  createRoom: (input: CreateRoomInput) => Promise<Room | null>;
  loading: boolean;
  error: string | null;
  clearError: () => void;
}

export function useCreateRoom(): UseCreateRoomResult {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearError = useCallback(() => setError(null), []);

  const createRoom = useCallback(async (input: CreateRoomInput): Promise<Room | null> => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API_BASE}/rooms`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        throw new Error(body?.error ?? `Failed to create room (${res.status})`);
      }
      const room: Room = await res.json();
      return room;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create room";
      setError(message);
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  return { createRoom, loading, error, clearError };
}

import { useCallback, useEffect, useRef, useState } from "react";
import type { Room } from "@/components/room-card";

const API_BASE = "/api";
const POLL_INTERVAL = 5000;

interface UseRoomsResult {
  rooms: Room[];
  loading: boolean;
  error: string | null;
  refresh: () => void;
}

export function useRooms(): UseRoomsResult {
  const [rooms, setRooms] = useState<Room[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchRooms = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/rooms`);
      if (!res.ok) {
        throw new Error(`Failed to fetch rooms: ${res.status}`);
      }
      const data: Room[] = await res.json();
      setRooms(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch rooms");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRooms();
    intervalRef.current = setInterval(fetchRooms, POLL_INTERVAL);
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [fetchRooms]);

  return { rooms, loading, error, refresh: fetchRooms };
}

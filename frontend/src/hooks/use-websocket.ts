import { useCallback, useEffect, useRef, useState } from "react";
import {
  type BackfillMessage,
  type ConnectionState,
  type Envelope,
  ReconnectingWS,
  type SessionPayload,
} from "@/lib/reconnecting-ws";

export interface UseWebSocketOptions {
  url: string;
  roomID: string;
  username?: string;
  onMessage?: (envelope: Envelope) => void;
  onHistoryBatch?: (messages: BackfillMessage[], hasMore: boolean) => void;
  maxRetries?: number;
}

export interface UseWebSocketReturn {
  state: ConnectionState;
  session: SessionPayload | null;
  send: (type: string, payload: unknown) => void;
  fetchHistory: (beforeID: string, limit?: number) => void;
  disconnect: () => void;
}

export function useWebSocket(
  opts: UseWebSocketOptions,
): UseWebSocketReturn {
  const [state, setState] = useState<ConnectionState>("disconnected");
  const [session, setSession] = useState<SessionPayload | null>(null);
  const wsRef = useRef<ReconnectingWS | null>(null);
  const onMessageRef = useRef(opts.onMessage);
  onMessageRef.current = opts.onMessage;
  const onHistoryBatchRef = useRef(opts.onHistoryBatch);
  onHistoryBatchRef.current = opts.onHistoryBatch;

  useEffect(() => {
    const ws = new ReconnectingWS({
      url: opts.url,
      roomID: opts.roomID,
      username: opts.username,
      maxRetries: opts.maxRetries,
      onStateChange: setState,
      onSession: setSession,
      onMessage: (env) => onMessageRef.current?.(env),
      onHistoryBatch: (msgs, hasMore) => onHistoryBatchRef.current?.(msgs, hasMore),
    });
    wsRef.current = ws;
    ws.connect();

    return () => {
      ws.disconnect();
      wsRef.current = null;
    };
    // Reconnect when url/room/username changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [opts.url, opts.roomID, opts.username]);

  const send = useCallback((type: string, payload: unknown) => {
    wsRef.current?.send(type, payload);
  }, []);

  const fetchHistory = useCallback((beforeID: string, limit?: number) => {
    wsRef.current?.fetchHistory(beforeID, limit);
  }, []);

  const disconnect = useCallback(() => {
    wsRef.current?.disconnect();
    wsRef.current = null;
  }, []);

  return { state, session, send, fetchHistory, disconnect };
}

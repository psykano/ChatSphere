import { useCallback, useEffect, useRef, useState } from "react";
import {
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
  maxRetries?: number;
}

export interface UseWebSocketReturn {
  state: ConnectionState;
  session: SessionPayload | null;
  send: (type: string, payload: unknown) => void;
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

  useEffect(() => {
    const ws = new ReconnectingWS({
      url: opts.url,
      roomID: opts.roomID,
      username: opts.username,
      maxRetries: opts.maxRetries,
      onStateChange: setState,
      onSession: setSession,
      onMessage: (env) => onMessageRef.current?.(env),
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

  const disconnect = useCallback(() => {
    wsRef.current?.disconnect();
    wsRef.current = null;
  }, []);

  return { state, session, send, disconnect };
}

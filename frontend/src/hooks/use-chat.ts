import { useCallback, useEffect, useRef, useState } from "react";
import type { BackfillMessage, Envelope, TypingPayload } from "@/lib/reconnecting-ws";
import { useWebSocket } from "@/hooks/use-websocket";

export interface ChatMessage {
  id: string;
  room_id: string;
  user_id?: string;
  username?: string;
  content: string;
  type: string;
  created_at: string;
}

export interface OnlineUser {
  user_id: string;
  username: string;
}

export interface MuteStatus {
  muted: boolean;
  expiresAt: string | null; // RFC3339 timestamp or null for permanent
}

interface UseChatOptions {
  roomID: string;
  username?: string;
}

function backfillToChat(msg: BackfillMessage): ChatMessage {
  return {
    id: msg.id,
    room_id: msg.room_id,
    user_id: msg.user_id,
    username: msg.username,
    content: msg.content,
    type: msg.type,
    created_at: msg.created_at,
  };
}

// How long before a typing indicator expires (ms).
const TYPING_TIMEOUT = 3000;

export function useChat({ roomID, username }: UseChatOptions) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [onlineUsers, setOnlineUsers] = useState<OnlineUser[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [typingUsers, setTypingUsers] = useState<Map<string, string>>(new Map());
  const [muteStatus, setMuteStatus] = useState<MuteStatus>({ muted: false, expiresAt: null });
  const loadingHistoryRef = useRef(false);
  const typingTimersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const muteTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clean up typing timers and mute timer on unmount.
  useEffect(() => {
    const timers = typingTimersRef.current;
    return () => {
      for (const timer of timers.values()) {
        clearTimeout(timer);
      }
      if (muteTimerRef.current) {
        clearTimeout(muteTimerRef.current);
      }
    };
  }, []);

  // Reset state when room or username changes (triggers WebSocket reconnect).
  useEffect(() => {
    setMessages([]);
    setOnlineUsers([]);
    setHasMore(false);
    setTypingUsers(new Map());
  }, [roomID, username]);

  const handleMessage = useCallback((envelope: Envelope) => {
    if (envelope.type === "message" || envelope.type === "chat") {
      const msg = envelope.payload as ChatMessage;
      setMessages((prev) => [...prev, msg]);
      // Clear typing indicator when user sends a message.
      if (msg.user_id) {
        setTypingUsers((prev) => {
          if (!prev.has(msg.user_id!)) return prev;
          const next = new Map(prev);
          next.delete(msg.user_id!);
          return next;
        });
        const timer = typingTimersRef.current.get(msg.user_id);
        if (timer) {
          clearTimeout(timer);
          typingTimersRef.current.delete(msg.user_id);
        }
      }
      return;
    }
    if (envelope.type === "system") {
      const msg = envelope.payload as ChatMessage;
      setMessages((prev) => [...prev, msg]);
      return;
    }
    if (envelope.type === "presence") {
      const payload = envelope.payload as { users: OnlineUser[] };
      setOnlineUsers(payload.users ?? []);
      return;
    }
    if (envelope.type === "join") {
      const msg = envelope.payload as ChatMessage;
      setMessages((prev) => [...prev, msg]);
      return;
    }
    if (envelope.type === "leave") {
      const msg = envelope.payload as ChatMessage;
      setMessages((prev) => [...prev, msg]);
      return;
    }
    if (envelope.type === "mute_status") {
      const payload = envelope.payload as { muted: boolean; expires_at?: string };
      if (muteTimerRef.current) {
        clearTimeout(muteTimerRef.current);
        muteTimerRef.current = null;
      }
      setMuteStatus({ muted: payload.muted, expiresAt: payload.expires_at ?? null });
      if (payload.muted && payload.expires_at) {
        const remaining = new Date(payload.expires_at).getTime() - Date.now();
        if (remaining > 0) {
          muteTimerRef.current = setTimeout(() => {
            muteTimerRef.current = null;
            setMuteStatus({ muted: false, expiresAt: null });
          }, remaining);
        } else {
          setMuteStatus({ muted: false, expiresAt: null });
        }
      }
      return;
    }
    if (envelope.type === "typing") {
      const payload = envelope.payload as TypingPayload;
      setTypingUsers((prev) => {
        const next = new Map(prev);
        next.set(payload.user_id, payload.username);
        return next;
      });
      // Clear any existing timer for this user.
      const existing = typingTimersRef.current.get(payload.user_id);
      if (existing) clearTimeout(existing);
      // Set a timer to remove the typing indicator.
      const timer = setTimeout(() => {
        typingTimersRef.current.delete(payload.user_id);
        setTypingUsers((prev) => {
          if (!prev.has(payload.user_id)) return prev;
          const next = new Map(prev);
          next.delete(payload.user_id);
          return next;
        });
      }, TYPING_TIMEOUT);
      typingTimersRef.current.set(payload.user_id, timer);
      return;
    }
  }, []);

  const handleHistoryBatch = useCallback(
    (batch: BackfillMessage[], more: boolean) => {
      setMessages((prev) => [...batch.map(backfillToChat), ...prev]);
      setHasMore(more);
      loadingHistoryRef.current = false;
    },
    [],
  );

  const handleBackfillGap = useCallback(() => {
    const gapMessage: ChatMessage = {
      id: `gap-${Date.now()}`,
      room_id: roomID,
      content: "Some messages may be missing",
      type: "gap",
      created_at: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, gapMessage]);
  }, [roomID]);

  const wsURL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws`;

  const { state, session, send, fetchHistory, disconnect, retry } = useWebSocket({
    url: wsURL,
    roomID,
    username,
    onMessage: handleMessage,
    onHistoryBatch: handleHistoryBatch,
    onBackfillGap: handleBackfillGap,
  });

  const sendMessage = useCallback(
    (content: string) => {
      if (!content.trim()) return;
      send("chat", { content: content.trim() });
    },
    [send],
  );

  const sendTyping = useCallback(() => {
    send("typing", {});
  }, [send]);

  const sendKick = useCallback(
    (userID: string) => {
      send("kick", { user_id: userID });
    },
    [send],
  );

  const sendBan = useCallback(
    (userID: string) => {
      send("ban", { user_id: userID });
    },
    [send],
  );

  const sendMute = useCallback(
    (userID: string) => {
      send("mute", { user_id: userID });
    },
    [send],
  );

  const loadMore = useCallback(() => {
    if (loadingHistoryRef.current || !hasMore || messages.length === 0) return;
    loadingHistoryRef.current = true;
    fetchHistory(messages[0].id);
  }, [fetchHistory, hasMore, messages]);

  return {
    messages,
    onlineUsers,
    typingUsers,
    muteStatus,
    connectionState: state,
    session,
    hasMore,
    sendMessage,
    sendTyping,
    sendKick,
    sendBan,
    sendMute,
    loadMore,
    disconnect,
    retry,
  };
}

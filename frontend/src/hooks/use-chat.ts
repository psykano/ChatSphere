import { useCallback, useRef, useState } from "react";
import type { BackfillMessage, Envelope } from "@/lib/reconnecting-ws";
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

export function useChat({ roomID, username }: UseChatOptions) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [onlineUsers, setOnlineUsers] = useState<OnlineUser[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const loadingHistoryRef = useRef(false);

  const handleMessage = useCallback((envelope: Envelope) => {
    if (envelope.type === "message") {
      const msg = envelope.payload as ChatMessage;
      setMessages((prev) => [...prev, msg]);
      return;
    }
    if (envelope.type === "system") {
      const msg = envelope.payload as ChatMessage;
      setMessages((prev) => [...prev, msg]);
      return;
    }
    if (envelope.type === "presence") {
      const users = envelope.payload as OnlineUser[];
      setOnlineUsers(users);
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
  }, []);

  const handleHistoryBatch = useCallback(
    (batch: BackfillMessage[], more: boolean) => {
      setMessages((prev) => [...batch.map(backfillToChat), ...prev]);
      setHasMore(more);
      loadingHistoryRef.current = false;
    },
    [],
  );

  const wsURL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws`;

  const { state, session, send, fetchHistory, disconnect } = useWebSocket({
    url: wsURL,
    roomID,
    username,
    onMessage: handleMessage,
    onHistoryBatch: handleHistoryBatch,
  });

  const sendMessage = useCallback(
    (content: string) => {
      if (!content.trim()) return;
      send("message", { content: content.trim() });
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
    connectionState: state,
    session,
    hasMore,
    sendMessage,
    loadMore,
    disconnect,
  };
}

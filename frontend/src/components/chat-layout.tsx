import { useEffect, useRef } from "react";
import type { Room } from "@/components/room-card";
import { useChat } from "@/hooks/use-chat";
import { ChatSidebar } from "@/components/chat-sidebar";
import { MessageBubble } from "@/components/message-bubble";
import { MessageInput } from "@/components/message-input";
import { ThemeToggle } from "@/components/theme-toggle";

interface ChatLayoutProps {
  room: Room;
  onLeave: () => void;
}

export function ChatLayout({ room, onLeave }: ChatLayoutProps) {
  const {
    messages,
    onlineUsers,
    connectionState,
    session,
    hasMore,
    sendMessage,
    loadMore,
    disconnect,
  } = useChat({ roomID: room.id });

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const prevMessageCountRef = useRef(0);

  // Auto-scroll when new messages arrive (only if already near bottom)
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const isNewMessage = messages.length > prevMessageCountRef.current;
    prevMessageCountRef.current = messages.length;

    if (!isNewMessage) return;

    const { scrollTop, scrollHeight, clientHeight } = container;
    const isNearBottom = scrollHeight - scrollTop - clientHeight < 100;

    if (isNearBottom) {
      messagesEndRef.current?.scrollIntoView?.({ behavior: "smooth" });
    }
  }, [messages.length]);

  // Scroll to bottom on initial load
  const isConnected = connectionState === "connected";
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView?.();
  }, [isConnected]);

  function handleLeave() {
    disconnect();
    onLeave();
  }

  function handleScroll() {
    const container = scrollContainerRef.current;
    if (!container) return;
    if (container.scrollTop === 0 && hasMore) {
      loadMore();
    }
  }

  return (
    <div className="flex h-screen" role="main">
      <ChatSidebar
        room={room}
        onlineUsers={onlineUsers}
        connectionState={connectionState}
        onLeave={handleLeave}
      />

      <div className="flex flex-1 flex-col">
        {/* Room header */}
        <header className="flex items-center border-b border-border bg-card px-6 py-3">
          <h1 className="truncate text-lg font-semibold text-card-foreground">
            # {room.name}
          </h1>
          <span className="ml-3 text-sm text-muted-foreground">
            {onlineUsers.length} online
          </span>
          <div className="ml-auto">
            <ThemeToggle />
          </div>
        </header>

        {/* Messages area */}
        <div
          ref={scrollContainerRef}
          onScroll={handleScroll}
          className="flex-1 overflow-y-auto px-6 py-4"
          role="log"
          aria-label="Chat messages"
        >
          {hasMore && (
            <button
              type="button"
              onClick={loadMore}
              className="mb-4 w-full text-center text-sm text-muted-foreground hover:text-foreground"
            >
              Load earlier messages
            </button>
          )}

          <div className="space-y-2">
            {messages.map((msg) => (
              <MessageBubble
                key={msg.id}
                message={msg}
                isOwn={msg.user_id === session?.user_id}
              />
            ))}
          </div>

          <div ref={messagesEndRef} />
        </div>

        {/* Input bar */}
        <MessageInput
          onSend={sendMessage}
          disabled={connectionState !== "connected"}
        />
      </div>
    </div>
  );
}

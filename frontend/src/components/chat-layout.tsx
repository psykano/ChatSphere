import { useEffect, useRef, useState } from "react";
import { Menu, X } from "lucide-react";
import type { Room } from "@/components/room-card";
import { useChat } from "@/hooks/use-chat";
import { ChatSidebar } from "@/components/chat-sidebar";
import { MessageBubble } from "@/components/message-bubble";
import { MessageInput } from "@/components/message-input";
import { TypingIndicator } from "@/components/typing-indicator";
import { UsernameInput } from "@/components/username-input";
import { ThemeToggle } from "@/components/theme-toggle";
import { ConnectionStatusBanner } from "@/components/connection-status-banner";
import { isSameUserAsPrevious } from "@/lib/message-grouping";

interface ChatLayoutProps {
  room: Room;
  onLeave: () => void;
}

export function ChatLayout({ room, onLeave }: ChatLayoutProps) {
  const [username, setUsername] = useState<string | undefined>();

  const {
    messages,
    onlineUsers,
    typingUsers,
    connectionState,
    session,
    hasMore,
    sendMessage,
    sendTyping,
    loadMore,
    disconnect,
    retry,
  } = useChat({ roomID: room.id, username });

  const [sidebarOpen, setSidebarOpen] = useState(false);
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
      {/* Mobile sidebar overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-30 bg-black/50 md:hidden"
          onClick={() => setSidebarOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar: always visible on md+, overlay drawer on mobile */}
      <div
        className={`fixed inset-y-0 left-0 z-40 w-60 transform transition-transform duration-200 md:static md:translate-x-0 ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full"
        }`}
      >
        <ChatSidebar
          room={room}
          onlineUsers={onlineUsers}
          connectionState={connectionState}
          onLeave={handleLeave}
        />
      </div>

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Room header */}
        <header className="flex items-center border-b border-border bg-card px-4 py-3 md:px-6">
          <button
            type="button"
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="mr-2 inline-flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:text-foreground md:hidden"
            aria-label={sidebarOpen ? "Close sidebar" : "Open sidebar"}
          >
            {sidebarOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
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

        {/* Connection status banner */}
        <ConnectionStatusBanner connectionState={connectionState} onRetry={retry} />

        {/* Username input bar */}
        {!username && (
          <UsernameInput onSubmit={setUsername} />
        )}

        {/* Messages area */}
        <div
          ref={scrollContainerRef}
          onScroll={handleScroll}
          className="flex-1 overflow-y-auto px-4 py-4 md:px-6"
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
            {messages.map((msg, idx) => {
              const grouped = isSameUserAsPrevious(messages, idx);

              return (
                <div key={msg.id} className={grouped ? "-mt-1" : ""}>
                  <MessageBubble
                    message={msg}
                    isOwn={msg.user_id === session?.user_id}
                    showUsername={!grouped}
                  />
                </div>
              );
            })}
          </div>

          <div ref={messagesEndRef} />
        </div>

        {/* Typing indicator */}
        <TypingIndicator typingUsers={typingUsers} />

        {/* Input bar */}
        <MessageInput
          onSend={sendMessage}
          onTyping={sendTyping}
          disabled={connectionState !== "connected"}
          readOnly={!username}
        />
      </div>
    </div>
  );
}

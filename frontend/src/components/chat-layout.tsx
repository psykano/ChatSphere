import { useEffect, useRef, useState } from "react";
import { Menu, X } from "lucide-react";
import type { Room } from "@/components/room-card";
import { useChat } from "@/hooks/use-chat";
import { ChatSidebar } from "@/components/chat-sidebar";
import { MessageBubble } from "@/components/message-bubble";
import { MessageInput, type MessageInputHandle } from "@/components/message-input";
import { TypingIndicator } from "@/components/typing-indicator";
import { UsernameInput } from "@/components/username-input";
import { ThemeToggle } from "@/components/theme-toggle";
import { ConnectionStatusBanner } from "@/components/connection-status-banner";
import { isSameUserAsPrevious } from "@/lib/message-grouping";
import { useDocumentTitle } from "@/hooks/use-document-title";

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
    muteStatus,
    connectionState,
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
  } = useChat({ roomID: room.id, username });

  const { incrementUnread } = useDocumentTitle();

  const [sidebarOpen, setSidebarOpen] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const prevMessageCountRef = useRef(0);
  const messageInputRef = useRef<MessageInputHandle>(null);

  // Track unread messages when tab is hidden
  const prevUnreadCountRef = useRef(messages.length);
  useEffect(() => {
    const newCount = messages.length - prevUnreadCountRef.current;
    prevUnreadCountRef.current = messages.length;
    for (let i = 0; i < newCount; i++) {
      incrementUnread();
    }
  }, [messages.length, incrementUnread]);

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

  // Scroll to bottom on initial load and after reconnects (e.g. username set)
  const isConnected = connectionState === "connected";
  const hasScrolledOnLoad = useRef(false);
  useEffect(() => {
    if (!isConnected) {
      hasScrolledOnLoad.current = false;
    }
    if (isConnected && messages.length > 0 && !hasScrolledOnLoad.current) {
      hasScrolledOnLoad.current = true;
      messagesEndRef.current?.scrollIntoView?.();
    }
  }, [isConnected, messages.length]);

  function handleLeave() {
    disconnect();
    onLeave();
  }

  function handleMention(mentionUsername: string) {
    messageInputRef.current?.insertText(`@${mentionUsername} `);
  }

  const isCreator = session?.is_creator ?? false;

  function findUserID(targetUsername: string): string | undefined {
    return onlineUsers.find((u) => u.username === targetUsername)?.user_id;
  }

  function handleKick(targetUsername: string) {
    const userID = findUserID(targetUsername);
    if (userID) sendKick(userID);
  }

  function handleBan(targetUsername: string) {
    const userID = findUserID(targetUsername);
    if (userID) sendBan(userID);
  }

  function handleMute(targetUsername: string) {
    const userID = findUserID(targetUsername);
    if (userID) sendMute(userID);
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
                    onMention={handleMention}
                    onKick={isCreator && msg.user_id !== session?.user_id ? handleKick : undefined}
                    onBan={isCreator && msg.user_id !== session?.user_id ? handleBan : undefined}
                    onMute={isCreator && msg.user_id !== session?.user_id ? handleMute : undefined}
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
          ref={messageInputRef}
          onSend={sendMessage}
          onTyping={sendTyping}
          disabled={connectionState !== "connected"}
          readOnly={!username}
          muteInfo={muteStatus}
        />
      </div>
    </div>
  );
}

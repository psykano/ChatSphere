import { cn } from "@/lib/utils";
import type { ChatMessage } from "@/hooks/use-chat";
import { MarkdownContent } from "./markdown-content";
import { UserContextMenu } from "./user-context-menu";

interface MessageBubbleProps {
  message: ChatMessage;
  isOwn: boolean;
  showUsername?: boolean;
  onMention?: (username: string) => void;
}

function formatTime(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function MessageBubble({ message, isOwn, showUsername = true, onMention }: MessageBubbleProps) {
  const isSystem =
    message.type === "system" ||
    message.type === "join" ||
    message.type === "leave";

  if (message.type === "gap") {
    return (
      <div className="flex items-center gap-2 py-2" role="separator">
        <div className="h-px flex-1 bg-border" />
        <span className="text-xs text-muted-foreground">
          {message.content}
        </span>
        <div className="h-px flex-1 bg-border" />
      </div>
    );
  }

  if (isSystem) {
    return (
      <div className="flex justify-center py-1">
        <span className="text-xs text-muted-foreground italic">
          {message.content}
        </span>
      </div>
    );
  }

  return (
    <div
      className={cn("flex flex-col gap-0.5 max-w-[85%] sm:max-w-[75%]", isOwn ? "ml-auto items-end" : "items-start")}
    >
      {!isOwn && message.username && showUsername && (
        <UserContextMenu username={message.username} onMention={onMention}>
          <span className="text-xs font-medium text-muted-foreground px-1">
            {message.username}
          </span>
        </UserContextMenu>
      )}
      <div
        className={cn(
          "rounded-lg px-3 py-2 text-sm break-words",
          isOwn
            ? "bg-primary text-primary-foreground"
            : "bg-secondary text-secondary-foreground",
        )}
      >
        <MarkdownContent content={message.content} />
      </div>
      <span className="text-[10px] text-muted-foreground px-1">
        {formatTime(message.created_at)}
      </span>
    </div>
  );
}

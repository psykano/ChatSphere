import { cn } from "@/lib/utils";
import type { ChatMessage } from "@/hooks/use-chat";

interface MessageBubbleProps {
  message: ChatMessage;
  isOwn: boolean;
}

function formatTime(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function MessageBubble({ message, isOwn }: MessageBubbleProps) {
  const isSystem =
    message.type === "system" ||
    message.type === "join" ||
    message.type === "leave";

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
      {!isOwn && message.username && (
        <span className="text-xs font-medium text-muted-foreground px-1">
          {message.username}
        </span>
      )}
      <div
        className={cn(
          "rounded-lg px-3 py-2 text-sm break-words",
          isOwn
            ? "bg-primary text-primary-foreground"
            : "bg-secondary text-secondary-foreground",
        )}
      >
        {message.content}
      </div>
      <span className="text-[10px] text-muted-foreground px-1">
        {formatTime(message.created_at)}
      </span>
    </div>
  );
}

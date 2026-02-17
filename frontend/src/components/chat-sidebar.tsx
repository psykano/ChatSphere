import type { Room } from "@/components/room-card";
import type { OnlineUser } from "@/hooks/use-chat";
import type { ConnectionState } from "@/lib/reconnecting-ws";
import { cn } from "@/lib/utils";

interface ChatSidebarProps {
  room: Room;
  onlineUsers: OnlineUser[];
  connectionState: ConnectionState;
  onLeave: () => void;
}

export function ChatSidebar({
  room,
  onlineUsers,
  connectionState,
  onLeave,
}: ChatSidebarProps) {
  return (
    <aside className="flex w-60 shrink-0 flex-col border-r border-border bg-card">
      <div className="border-b border-border p-4">
        <h2 className="truncate font-semibold text-card-foreground">
          {room.name}
        </h2>
        {room.description && (
          <p className="mt-1 text-xs text-muted-foreground line-clamp-2">
            {room.description}
          </p>
        )}
        <div className="mt-2 flex items-center gap-1.5 text-xs text-muted-foreground">
          <span
            className={cn(
              "inline-block h-2 w-2 rounded-full",
              connectionState === "connected"
                ? "bg-green-500"
                : connectionState === "reconnecting" || connectionState === "connecting"
                  ? "bg-yellow-500"
                  : "bg-muted-foreground",
            )}
            aria-hidden="true"
          />
          <span>
            {connectionState === "connected"
              ? "Connected"
              : connectionState === "reconnecting"
                ? "Reconnecting..."
                : connectionState === "connecting"
                  ? "Connecting..."
                  : "Disconnected"}
          </span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4">
        <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
          Online — {onlineUsers.length}
        </h3>
        <ul className="mt-2 space-y-1" aria-label="Online users">
          {onlineUsers.map((user) => (
            <li
              key={user.user_id}
              className="flex items-center gap-2 rounded px-2 py-1 text-sm text-card-foreground"
            >
              <span
                className="inline-block h-2 w-2 shrink-0 rounded-full bg-green-500"
                aria-hidden="true"
              />
              <span className="truncate">{user.username}</span>
            </li>
          ))}
        </ul>
      </div>

      <div className="border-t border-border p-4">
        <button
          type="button"
          onClick={onLeave}
          className="w-full rounded-md bg-secondary px-3 py-2 text-sm text-secondary-foreground transition-colors hover:bg-secondary/80"
        >
          Leave Room
        </button>
      </div>
    </aside>
  );
}

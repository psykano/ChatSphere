import { cn } from "@/lib/utils";

export interface LastMessage {
  content: string;
  username?: string;
  created_at: string;
}

export interface Room {
  id: string;
  name: string;
  description?: string;
  capacity: number;
  active_users: number;
  creator_id?: string;
  last_message?: LastMessage;
  created_at: string;
}

interface RoomCardProps {
  room: Room;
  onClick?: (room: Room) => void;
}

function formatTimeAgo(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const seconds = Math.floor((now - then) / 1000);

  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export function RoomCard({ room, onClick }: RoomCardProps) {
  const occupancyPercent = Math.round((room.active_users / room.capacity) * 100);

  return (
    <button
      type="button"
      onClick={() => onClick?.(room)}
      className="w-full text-left rounded-lg border border-border bg-card p-4 transition-colors hover:bg-accent"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <h3 className="truncate font-semibold text-card-foreground">
            {room.name}
          </h3>
          {room.description && (
            <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">
              {room.description}
            </p>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-1.5 text-sm text-muted-foreground">
          <span
            className={cn(
              "inline-block h-2 w-2 rounded-full",
              room.active_users > 0 ? "bg-green-500" : "bg-muted-foreground"
            )}
            aria-hidden="true"
          />
          <span>
            {room.active_users}/{room.capacity}
          </span>
        </div>
      </div>

      {room.last_message && (
        <p className="mt-2 truncate text-sm text-muted-foreground">
          {room.last_message.username && (
            <span className="font-medium text-card-foreground">
              {room.last_message.username}:{" "}
            </span>
          )}
          {room.last_message.content}
        </p>
      )}

      <div className="mt-3 flex items-center justify-between">
        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-secondary">
          <div
            className="h-full rounded-full bg-primary transition-all"
            style={{ width: `${Math.min(occupancyPercent, 100)}%` }}
          />
        </div>
        {room.creator_id && (
          <span className="ml-3 shrink-0 text-xs text-muted-foreground">
            by {room.creator_id}
          </span>
        )}
      </div>

      {room.last_message && (
        <span className="mt-1 block text-xs text-muted-foreground">
          {formatTimeAgo(room.last_message.created_at)}
        </span>
      )}
    </button>
  );
}

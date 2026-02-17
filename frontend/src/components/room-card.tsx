import { cn } from "@/lib/utils";

export interface Room {
  id: string;
  name: string;
  description?: string;
  capacity: number;
  active_users: number;
  created_at: string;
}

interface RoomCardProps {
  room: Room;
  onClick?: (room: Room) => void;
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
      <div className="mt-3">
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary">
          <div
            className="h-full rounded-full bg-primary transition-all"
            style={{ width: `${Math.min(occupancyPercent, 100)}%` }}
          />
        </div>
      </div>
    </button>
  );
}

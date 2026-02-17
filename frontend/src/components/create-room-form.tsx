import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface CreateRoomFormProps {
  onSubmit: (input: {
    name: string;
    description: string;
    capacity: number;
    public: boolean;
  }) => void;
  loading?: boolean;
  error?: string | null;
}

export function CreateRoomForm({ onSubmit, loading, error }: CreateRoomFormProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [capacity, setCapacity] = useState("50");
  const [isPublic, setIsPublic] = useState(true);

  const trimmedName = name.trim();
  const capacityNum = parseInt(capacity, 10);
  const isValidName = trimmedName.length > 0 && trimmedName.length <= 100;
  const isValidCapacity = !isNaN(capacityNum) && capacityNum >= 2 && capacityNum <= 100;
  const isValid = isValidName && isValidCapacity && description.length <= 500;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (isValid && !loading) {
      onSubmit({
        name: trimmedName,
        description: description.trim(),
        capacity: capacityNum,
        public: isPublic,
      });
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="space-y-1.5">
        <label htmlFor="room-name" className="text-sm font-medium text-foreground">
          Room name
        </label>
        <input
          id="room-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value.slice(0, 100))}
          placeholder="Give your room a name"
          maxLength={100}
          className={cn(
            "h-10 w-full rounded-md border bg-background px-3 text-sm",
            "placeholder:text-muted-foreground",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            "border-input"
          )}
        />
      </div>

      <div className="space-y-1.5">
        <label htmlFor="room-description" className="text-sm font-medium text-foreground">
          Description <span className="font-normal text-muted-foreground">(optional)</span>
        </label>
        <textarea
          id="room-description"
          value={description}
          onChange={(e) => setDescription(e.target.value.slice(0, 500))}
          placeholder="What's this room about?"
          maxLength={500}
          rows={2}
          className={cn(
            "w-full rounded-md border bg-background px-3 py-2 text-sm",
            "placeholder:text-muted-foreground",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            "resize-none border-input"
          )}
        />
      </div>

      <div className="flex gap-4">
        <div className="flex-1 space-y-1.5">
          <label htmlFor="room-capacity" className="text-sm font-medium text-foreground">
            Capacity
          </label>
          <input
            id="room-capacity"
            type="number"
            value={capacity}
            onChange={(e) => setCapacity(e.target.value)}
            min={2}
            max={100}
            className={cn(
              "h-10 w-full rounded-md border bg-background px-3 text-sm",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              !isValidCapacity && capacity !== "" ? "border-destructive-foreground" : "border-input"
            )}
          />
        </div>

        <div className="flex-1 space-y-1.5">
          <span className="text-sm font-medium text-foreground">Visibility</span>
          <button
            type="button"
            role="switch"
            aria-checked={isPublic}
            aria-label="Public room"
            onClick={() => setIsPublic((prev) => !prev)}
            className="flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 text-sm"
          >
            <span className="text-muted-foreground">
              {isPublic ? "Public" : "Private"}
            </span>
            <span
              className={cn(
                "relative inline-flex h-5 w-9 shrink-0 rounded-full transition-colors",
                isPublic ? "bg-primary" : "bg-muted"
              )}
            >
              <span
                className={cn(
                  "inline-block h-4 w-4 rounded-full bg-background transition-transform mt-0.5",
                  isPublic ? "translate-x-4 ml-0.5" : "translate-x-0 ml-0.5"
                )}
              />
            </span>
          </button>
        </div>
      </div>

      <Button type="submit" disabled={!isValid || loading} className="w-full">
        {loading ? "Creating..." : "Create Room"}
      </Button>

      {error && (
        <p className="text-sm text-destructive-foreground" role="alert">
          {error}
        </p>
      )}
    </form>
  );
}

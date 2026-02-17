import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface EnterCodeBarProps {
  onJoin: (code: string) => void;
  loading?: boolean;
  error?: string | null;
}

export function EnterCodeBar({ onJoin, loading, error }: EnterCodeBarProps) {
  const [code, setCode] = useState("");

  const trimmed = code.trim().toUpperCase();
  const isValid = /^[A-Z0-9]{6}$/.test(trimmed);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (isValid && !loading) {
      onJoin(trimmed);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-2">
      <div className="flex gap-2">
        <input
          type="text"
          value={code}
          onChange={(e) => setCode(e.target.value.slice(0, 6))}
          placeholder="Enter room code"
          maxLength={6}
          aria-label="Room code"
          className={cn(
            "h-10 flex-1 rounded-md border bg-background px-3 text-sm tracking-widest uppercase",
            "placeholder:tracking-normal placeholder:normal-case placeholder:text-muted-foreground",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
            error ? "border-destructive-foreground" : "border-input"
          )}
        />
        <Button type="submit" disabled={!isValid || loading} size="default">
          {loading ? "Joining..." : "Join"}
        </Button>
      </div>
      {error && (
        <p className="text-sm text-destructive-foreground" role="alert">
          {error}
        </p>
      )}
    </form>
  );
}

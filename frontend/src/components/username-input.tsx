import { useState } from "react";
import { UserRound } from "lucide-react";

interface UsernameInputProps {
  onSubmit: (username: string) => void;
}

export function UsernameInput({ onSubmit }: UsernameInputProps) {
  const [value, setValue] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed) return;
    onSubmit(trimmed);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSubmit(e);
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-center gap-2 border-b border-border bg-card px-4 py-3 md:px-6"
      aria-label="Set username"
    >
      <UserRound className="h-4 w-4 shrink-0 text-muted-foreground" />
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Enter a display name..."
        maxLength={32}
        aria-label="Username"
        className="flex-1 bg-transparent text-sm placeholder:text-muted-foreground focus-visible:outline-none"
      />
      <button
        type="submit"
        disabled={!value.trim()}
        className="rounded-md bg-primary px-3 py-1 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
      >
        Join
      </button>
    </form>
  );
}

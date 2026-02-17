interface TypingIndicatorProps {
  typingUsers: Map<string, string>;
}

export function TypingIndicator({ typingUsers }: TypingIndicatorProps) {
  if (typingUsers.size === 0) return null;

  const names = Array.from(typingUsers.values());
  let text: string;

  if (names.length === 1) {
    text = `${names[0]} is typing...`;
  } else if (names.length === 2) {
    text = `${names[0]} and ${names[1]} are typing...`;
  } else {
    text = "Several people are typing...";
  }

  return (
    <div
      className="px-4 py-1 text-xs text-muted-foreground md:px-6"
      aria-live="polite"
      aria-label="Typing indicator"
    >
      {text}
    </div>
  );
}

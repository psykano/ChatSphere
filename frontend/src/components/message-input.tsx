import { useState, useRef, useCallback } from "react";
import { SendHorizontal } from "lucide-react";
import { EmojiPicker } from "./emoji-picker";

// Minimum interval between typing events (ms).
const TYPING_THROTTLE = 2000;

interface MessageInputProps {
  onSend: (content: string) => void;
  onTyping?: () => void;
  disabled?: boolean;
  readOnly?: boolean;
}

export function MessageInput({ onSend, onTyping, disabled, readOnly }: MessageInputProps) {
  const [value, setValue] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lastTypingSentRef = useRef(0);

  const emitTyping = useCallback(() => {
    if (!onTyping) return;
    const now = Date.now();
    if (now - lastTypingSentRef.current >= TYPING_THROTTLE) {
      lastTypingSentRef.current = now;
      onTyping();
    }
  }, [onTyping]);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed || disabled || readOnly) return;
    onSend(trimmed);
    setValue("");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  }

  function handleEmojiSelect(emoji: string) {
    const textarea = textareaRef.current;
    if (!textarea) {
      setValue((prev) => prev + emoji);
      return;
    }
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const newValue = value.slice(0, start) + emoji + value.slice(end);
    setValue(newValue);
    requestAnimationFrame(() => {
      const cursorPos = start + emoji.length;
      textarea.setSelectionRange(cursorPos, cursorPos);
      textarea.focus();
    });
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-end gap-2 border-t border-border bg-card p-3 sm:p-4"
    >
      <textarea
        ref={textareaRef}
        value={value}
        onChange={(e) => {
          setValue(e.target.value);
          if (e.target.value.trim()) emitTyping();
        }}
        onKeyDown={handleKeyDown}
        placeholder={readOnly ? "Set a username to start chatting" : "Type a message..."}
        disabled={disabled || readOnly}
        rows={1}
        aria-label="Message input"
        className="flex-1 resize-none rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
      />
      <EmojiPicker onSelect={handleEmojiSelect} disabled={disabled || readOnly} />
      <button
        type="submit"
        disabled={disabled || readOnly || !value.trim()}
        aria-label="Send message"
        className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
      >
        <SendHorizontal className="h-4 w-4" />
      </button>
    </form>
  );
}

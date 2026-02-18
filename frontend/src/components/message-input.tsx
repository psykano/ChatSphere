import { useState, useRef, useMemo, useCallback, useEffect, useImperativeHandle, forwardRef } from "react";
import { SendHorizontal } from "lucide-react";
import { EmojiPicker } from "./emoji-picker";

// Minimum interval between typing events (ms).
const TYPING_THROTTLE = 2000;

export interface MessageInputHandle {
  insertText: (text: string) => void;
}

export interface MuteInfo {
  muted: boolean;
  expiresAt: string | null;
}

interface MessageInputProps {
  onSend: (content: string) => void;
  onTyping?: () => void;
  disabled?: boolean;
  readOnly?: boolean;
  muteInfo?: MuteInfo;
}

function formatMuteRemaining(expiresAt: string): string {
  const remaining = Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / 1000));
  if (remaining <= 0) return "";
  const minutes = Math.ceil(remaining / 60);
  if (minutes === 1) return "1 minute";
  return `${minutes} minutes`;
}

function computeMuteLabel(muteInfo?: MuteInfo): string {
  if (!muteInfo?.muted) return "";
  if (!muteInfo.expiresAt) return "You have been muted";
  const text = formatMuteRemaining(muteInfo.expiresAt);
  return text ? `You have been muted for ${text}` : "";
}

export const MessageInput = forwardRef<MessageInputHandle, MessageInputProps>(function MessageInput({ onSend, onTyping, disabled, readOnly, muteInfo }, ref) {
  const [value, setValue] = useState("");
  const [tick, setTick] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const lastTypingSentRef = useRef(0);

  // Tick every second while muted with a timed expiry to update the countdown.
  useEffect(() => {
    if (!muteInfo?.muted || !muteInfo?.expiresAt) return;
    const interval = setInterval(() => setTick((t) => t + 1), 1000);
    return () => clearInterval(interval);
  }, [muteInfo?.muted, muteInfo?.expiresAt]);

  // eslint-disable-next-line react-hooks/exhaustive-deps -- tick triggers recomputation of time-based label
  const muteLabel = useMemo(() => computeMuteLabel(muteInfo), [muteInfo, tick]);

  useImperativeHandle(ref, () => ({
    insertText(text: string) {
      const textarea = textareaRef.current;
      if (!textarea) {
        setValue((prev) => prev + text);
        return;
      }
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const newValue = value.slice(0, start) + text + value.slice(end);
      setValue(newValue);
      requestAnimationFrame(() => {
        const cursorPos = start + text.length;
        textarea.setSelectionRange(cursorPos, cursorPos);
        textarea.focus();
      });
    },
  }), [value]);

  const emitTyping = useCallback(() => {
    if (!onTyping) return;
    const now = Date.now();
    if (now - lastTypingSentRef.current >= TYPING_THROTTLE) {
      lastTypingSentRef.current = now;
      onTyping();
    }
  }, [onTyping]);

  const isMuted = muteInfo?.muted ?? false;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed || disabled || readOnly || isMuted) return;
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

  const inputDisabled = disabled || readOnly || isMuted;
  const placeholder = readOnly
    ? "Set a username to start chatting"
    : isMuted
      ? "You are muted"
      : "Type a message...";

  return (
    <div>
      {isMuted && muteLabel && (
        <div
          role="alert"
          className="border-t border-border bg-destructive/10 px-3 py-2 text-center text-sm font-medium text-destructive sm:px-4"
        >
          {muteLabel}
        </div>
      )}
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
          placeholder={placeholder}
          disabled={inputDisabled}
          rows={1}
          aria-label="Message input"
          className="flex-1 resize-none rounded-md border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
        />
        <EmojiPicker onSelect={handleEmojiSelect} disabled={inputDisabled} />
        <button
          type="submit"
          disabled={inputDisabled || !value.trim()}
          aria-label="Send message"
          className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
        >
          <SendHorizontal className="h-4 w-4" />
        </button>
      </form>
    </div>
  );
});

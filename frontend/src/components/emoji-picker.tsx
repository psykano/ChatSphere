import { useState, useRef, useEffect } from "react";
import { Smile } from "lucide-react";

const EMOJI_CATEGORIES: { name: string; emojis: string[] }[] = [
  {
    name: "Smileys",
    emojis: [
      "😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂",
      "🙂", "😊", "😇", "🥰", "😍", "🤩", "😘", "😋",
      "😛", "😜", "🤪", "😎", "🤗", "🤔", "🤭", "😐",
      "😑", "😶", "🙄", "😏", "😬", "🤥", "😌", "😴",
    ],
  },
  {
    name: "Gestures",
    emojis: [
      "👋", "🤚", "🖐️", "✋", "👌", "🤌", "✌️", "🤞",
      "🤟", "🤘", "🤙", "👈", "👉", "👆", "👇", "👍",
      "👎", "👏", "🙌", "🤝", "🙏", "💪", "🫶", "❤️",
    ],
  },
  {
    name: "Animals",
    emojis: [
      "🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼",
      "🐨", "🐯", "🦁", "🐮", "🐷", "🐸", "🐵", "🐔",
    ],
  },
  {
    name: "Food",
    emojis: [
      "🍎", "🍕", "🍔", "🌮", "🍣", "🍜", "🍩", "🎂",
      "☕", "🍺", "🥤", "🧁", "🍫", "🍿", "🥑", "🍉",
    ],
  },
  {
    name: "Objects",
    emojis: [
      "🔥", "⭐", "🎉", "🎊", "💯", "💡", "🎵", "🎶",
      "💬", "💭", "🚀", "✨", "🏆", "🎯", "🎮", "📱",
    ],
  },
];

interface EmojiPickerProps {
  onSelect: (emoji: string) => void;
  disabled?: boolean;
}

export function EmojiPicker({ onSelect, disabled }: EmojiPickerProps) {
  const [open, setOpen] = useState(false);
  const [activeCategory, setActiveCategory] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function handleClickOutside(e: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function handleEscape(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open]);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        disabled={disabled}
        aria-label="Open emoji picker"
        className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
      >
        <Smile className="h-4 w-4" />
      </button>
      {open && (
        <div
          role="dialog"
          aria-label="Emoji picker"
          className="absolute bottom-full right-0 mb-2 w-72 rounded-lg border border-border bg-popover p-2 shadow-lg"
        >
          <div className="flex gap-1 border-b border-border pb-2 mb-2">
            {EMOJI_CATEGORIES.map((cat, i) => (
              <button
                key={cat.name}
                type="button"
                onClick={() => setActiveCategory(i)}
                aria-label={cat.name}
                className={`flex-1 rounded px-1 py-1 text-xs transition-colors ${
                  i === activeCategory
                    ? "bg-accent text-accent-foreground font-medium"
                    : "text-muted-foreground hover:bg-accent/50"
                }`}
              >
                {cat.emojis[0]}
              </button>
            ))}
          </div>
          <div className="grid grid-cols-8 gap-0.5 max-h-48 overflow-y-auto">
            {EMOJI_CATEGORIES[activeCategory].emojis.map((emoji) => (
              <button
                key={emoji}
                type="button"
                onClick={() => {
                  onSelect(emoji);
                  setOpen(false);
                }}
                className="flex h-8 w-8 items-center justify-center rounded text-lg hover:bg-accent transition-colors"
              >
                {emoji}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

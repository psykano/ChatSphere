import { useState, useRef, useEffect, useCallback } from "react";
import { AtSign, Copy, MessageSquare, UserX, ShieldBan, VolumeOff } from "lucide-react";

function clampToViewport(el: HTMLElement, x: number, y: number): { x: number; y: number } {
  const rect = el.getBoundingClientRect();
  const vw = window.innerWidth;
  const vh = window.innerHeight;

  let cx = x;
  let cy = y;

  if (cx + rect.width > vw) cx = vw - rect.width - 8;
  if (cy + rect.height > vh) cy = vh - rect.height - 8;
  if (cx < 0) cx = 8;
  if (cy < 0) cy = 8;

  return { x: cx, y: cy };
}

interface Position {
  x: number;
  y: number;
}

interface UserContextMenuProps {
  username: string;
  children: React.ReactNode;
  onMention?: (username: string) => void;
  onKick?: (username: string) => void;
  onBan?: (username: string) => void;
  onMute?: (username: string) => void;
}

export function UserContextMenu({ username, children, onMention, onKick, onBan, onMute }: UserContextMenuProps) {
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState<Position>({ x: 0, y: 0 });
  const [copied, setCopied] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLSpanElement>(null);

  const close = useCallback(() => {
    setOpen(false);
    setCopied(false);
  }, []);

  useEffect(() => {
    if (!open) return;
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        close();
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open, close]);

  useEffect(() => {
    if (!open) return;
    function handleEscape(e: KeyboardEvent) {
      if (e.key === "Escape") close();
    }
    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [open, close]);

  const menuRefCallback = useCallback((node: HTMLDivElement | null) => {
    (menuRef as React.MutableRefObject<HTMLDivElement | null>).current = node;
    if (node) {
      const clamped = clampToViewport(node, position.x, position.y);
      node.style.left = `${clamped.x}px`;
      node.style.top = `${clamped.y}px`;
    }
  }, [position]);

  function openMenu(x: number, y: number) {
    setPosition({ x, y });
    setOpen(true);
    setCopied(false);
  }

  function handleClick(e: React.MouseEvent) {
    e.preventDefault();
    openMenu(e.clientX, e.clientY);
  }

  function handleContextMenu(e: React.MouseEvent) {
    e.preventDefault();
    openMenu(e.clientX, e.clientY);
  }

  function handleCopyUsername() {
    navigator.clipboard.writeText(username);
    setCopied(true);
    setTimeout(close, 600);
  }

  function handleMention() {
    onMention?.(username);
    close();
  }

  return (
    <>
      <span
        ref={triggerRef}
        onClick={handleClick}
        onContextMenu={handleContextMenu}
        className="cursor-pointer hover:underline"
        role="button"
        tabIndex={0}
        aria-label={`Actions for ${username}`}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            const rect = (e.target as HTMLElement).getBoundingClientRect();
            openMenu(rect.left, rect.bottom + 4);
          }
        }}
      >
        {children}
      </span>

      {open && (
        <div
          ref={menuRefCallback}
          role="menu"
          aria-label={`User menu for ${username}`}
          className="fixed z-50 min-w-[160px] rounded-lg border border-border bg-popover p-1 shadow-lg"
          style={{ left: position.x, top: position.y }}
        >
          <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
            {username}
          </div>

          <div className="h-px bg-border my-1" role="separator" />

          {onMention && (
            <button
              type="button"
              role="menuitem"
              onClick={handleMention}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
            >
              <AtSign className="h-3.5 w-3.5" />
              Mention
            </button>
          )}

          <button
            type="button"
            role="menuitem"
            onClick={handleCopyUsername}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
          >
            <Copy className="h-3.5 w-3.5" />
            {copied ? "Copied!" : "Copy username"}
          </button>

          <button
            type="button"
            role="menuitem"
            onClick={() => {
              navigator.clipboard.writeText(`@${username}`);
              setCopied(true);
              setTimeout(close, 600);
            }}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
          >
            <MessageSquare className="h-3.5 w-3.5" />
            Copy @mention
          </button>

          {(onKick || onMute || onBan) && (
            <div className="h-px bg-border my-1" role="separator" />
          )}

          {onMute && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onMute(username);
                close();
              }}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-popover-foreground hover:bg-accent hover:text-accent-foreground transition-colors"
            >
              <VolumeOff className="h-3.5 w-3.5" />
              Mute
            </button>
          )}

          {onKick && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onKick(username);
                close();
              }}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-destructive hover:bg-destructive/10 transition-colors"
            >
              <UserX className="h-3.5 w-3.5" />
              Kick
            </button>
          )}

          {onBan && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onBan(username);
                close();
              }}
              className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-destructive hover:bg-destructive/10 transition-colors"
            >
              <ShieldBan className="h-3.5 w-3.5" />
              Ban
            </button>
          )}
        </div>
      )}
    </>
  );
}

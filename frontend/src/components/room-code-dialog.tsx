import { useState } from "react";
import { Button } from "@/components/ui/button";

interface RoomCodeDialogProps {
  roomName: string;
  code: string;
  onClose: () => void;
}

export function RoomCodeDialog({ roomName, code, onClose }: RoomCodeDialogProps) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback: select the code text
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      role="dialog"
      aria-label="Private room created"
    >
      <div className="mx-4 w-full max-w-sm rounded-lg border border-border bg-card p-6 shadow-lg">
        <h2 className="text-lg font-semibold text-card-foreground">
          Private room created
        </h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Share this code so others can join{" "}
          <span className="font-medium text-card-foreground">{roomName}</span>:
        </p>
        <div className="mt-4 flex items-center gap-2">
          <code className="flex-1 rounded-md bg-muted px-4 py-3 text-center text-2xl font-mono font-bold tracking-widest text-card-foreground">
            {code}
          </code>
          <Button variant="outline" size="sm" onClick={handleCopy}>
            {copied ? "Copied!" : "Copy"}
          </Button>
        </div>
        <Button className="mt-4 w-full" onClick={onClose}>
          Done
        </Button>
      </div>
    </div>
  );
}

import { WifiOff, Loader2 } from "lucide-react";
import type { ConnectionState } from "@/lib/reconnecting-ws";

interface ConnectionStatusBannerProps {
  connectionState: ConnectionState;
  onRetry?: () => void;
}

export function ConnectionStatusBanner({
  connectionState,
  onRetry,
}: ConnectionStatusBannerProps) {
  if (connectionState === "connected") return null;

  if (connectionState === "reconnecting" || connectionState === "connecting") {
    return (
      <div
        className="flex items-center justify-center gap-2 bg-yellow-500/15 px-4 py-2 text-sm text-yellow-400"
        role="status"
        aria-live="polite"
      >
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
        <span>
          {connectionState === "reconnecting"
            ? "Connection lost. Reconnecting..."
            : "Connecting..."}
        </span>
      </div>
    );
  }

  return (
    <div
      className="flex items-center justify-center gap-2 bg-red-500/15 px-4 py-2 text-sm text-red-400"
      role="alert"
      aria-live="assertive"
    >
      <WifiOff className="h-4 w-4" aria-hidden="true" />
      <span>Disconnected from server</span>
      {onRetry && (
        <button
          onClick={onRetry}
          className="ml-2 rounded bg-red-500/20 px-2 py-0.5 text-sm text-red-300 hover:bg-red-500/30 transition-colors"
        >
          Retry
        </button>
      )}
    </div>
  );
}

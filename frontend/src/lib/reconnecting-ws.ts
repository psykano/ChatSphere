export type ConnectionState =
  | "disconnected"
  | "connecting"
  | "connected"
  | "reconnecting";

export interface Envelope {
  type: string;
  payload: unknown;
}

export interface SessionPayload {
  session_id: string;
  user_id: string;
  username: string;
  resumed: boolean;
}

export interface BackfillMessage {
  id: string;
  room_id: string;
  user_id?: string;
  username?: string;
  content: string;
  type: string;
  created_at: string;
}

export interface ReconnectingWSOptions {
  url: string;
  roomID: string;
  username?: string;
  onMessage?: (envelope: Envelope) => void;
  onStateChange?: (state: ConnectionState) => void;
  onSession?: (session: SessionPayload) => void;
  maxRetries?: number;
  baseDelay?: number;
  maxDelay?: number;
}

const DEFAULT_MAX_RETRIES = 10;
const DEFAULT_BASE_DELAY = 500;
const DEFAULT_MAX_DELAY = 30_000;

export class ReconnectingWS {
  private ws: WebSocket | null = null;
  private sessionID: string | null = null;
  private state: ConnectionState = "disconnected";
  private retryCount = 0;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private disposed = false;

  private readonly opts: Required<
    Pick<ReconnectingWSOptions, "maxRetries" | "baseDelay" | "maxDelay">
  > &
    ReconnectingWSOptions;

  constructor(opts: ReconnectingWSOptions) {
    this.opts = {
      maxRetries: DEFAULT_MAX_RETRIES,
      baseDelay: DEFAULT_BASE_DELAY,
      maxDelay: DEFAULT_MAX_DELAY,
      ...opts,
    };
  }

  connect(): void {
    if (this.disposed) return;
    this.setState(this.retryCount > 0 ? "reconnecting" : "connecting");
    this.openSocket();
  }

  send(type: string, payload: unknown): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    this.ws.send(JSON.stringify({ type, payload }));
  }

  disconnect(): void {
    this.disposed = true;
    this.clearRetryTimer();
    if (this.ws) {
      this.ws.close(1000, "client disconnect");
      this.ws = null;
    }
    this.setState("disconnected");
  }

  getState(): ConnectionState {
    return this.state;
  }

  getSessionID(): string | null {
    return this.sessionID;
  }

  private openSocket(): void {
    if (this.disposed) return;

    const ws = new WebSocket(this.opts.url);
    this.ws = ws;

    ws.onopen = () => {
      if (this.disposed) {
        ws.close();
        return;
      }
      this.sendJoin();
    };

    ws.onmessage = (event: MessageEvent) => {
      if (this.disposed) return;

      let envelope: Envelope;
      try {
        envelope = JSON.parse(event.data as string) as Envelope;
      } catch {
        return;
      }

      if (envelope.type === "session") {
        const session = envelope.payload as SessionPayload;
        this.sessionID = session.session_id;
        this.retryCount = 0;
        this.setState("connected");
        this.opts.onSession?.(session);
        return;
      }

      if (envelope.type === "history" || envelope.type === "backfill") {
        const messages = envelope.payload as BackfillMessage[];
        for (const msg of messages) {
          this.opts.onMessage?.({ type: msg.type, payload: msg });
        }
        return;
      }

      this.opts.onMessage?.(envelope);
    };

    ws.onclose = (event: CloseEvent) => {
      if (this.disposed) return;

      // Server-initiated policy violation means the join was rejected (bad room, etc.).
      // Don't retry in that case.
      if (event.code === 1008) {
        this.setState("disconnected");
        return;
      }

      this.scheduleReconnect();
    };

    ws.onerror = () => {
      // onclose will fire after onerror — reconnect logic is there.
    };
  }

  private sendJoin(): void {
    const payload: Record<string, string> = { room_id: this.opts.roomID };
    if (this.opts.username) {
      payload.username = this.opts.username;
    }
    if (this.sessionID) {
      payload.session_id = this.sessionID;
    }
    this.send("join", payload);
  }

  private scheduleReconnect(): void {
    if (this.disposed) return;
    if (this.retryCount >= this.opts.maxRetries) {
      this.setState("disconnected");
      return;
    }

    this.setState("reconnecting");
    const delay = Math.min(
      this.opts.baseDelay * Math.pow(2, this.retryCount),
      this.opts.maxDelay,
    );
    // Add jitter: ±25% of the delay.
    const jitter = delay * 0.25 * (Math.random() * 2 - 1);
    this.retryCount++;

    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      this.openSocket();
    }, delay + jitter);
  }

  private clearRetryTimer(): void {
    if (this.retryTimer !== null) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }

  private setState(state: ConnectionState): void {
    if (this.state === state) return;
    this.state = state;
    this.opts.onStateChange?.(state);
  }
}

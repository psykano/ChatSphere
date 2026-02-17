import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ReconnectingWS, type ConnectionState, type SessionPayload } from "./reconnecting-ws";

// --- Mock WebSocket ---

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  static OPEN = 1;
  static CLOSED = 3;

  url: string;
  readyState = MockWebSocket.OPEN;
  onopen: ((ev: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;

  sent: string[] = [];
  closeCalled = false;

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.closeCalled = true;
    this.readyState = MockWebSocket.CLOSED;
  }

  // Test helpers to simulate server behavior.
  simulateOpen() {
    this.readyState = MockWebSocket.OPEN;
    this.onopen?.(new Event("open"));
  }

  simulateMessage(data: unknown) {
    this.onmessage?.(new MessageEvent("message", { data: JSON.stringify(data) }));
  }

  simulateClose(code = 1006) {
    this.readyState = MockWebSocket.CLOSED;
    this.onclose?.(new CloseEvent("close", { code }));
  }
}

beforeEach(() => {
  MockWebSocket.instances = [];
  vi.stubGlobal("WebSocket", MockWebSocket);
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

function lastSocket(): MockWebSocket {
  return MockWebSocket.instances[MockWebSocket.instances.length - 1];
}

function sessionEnvelope(overrides: Partial<SessionPayload> = {}): object {
  return {
    type: "session",
    payload: {
      session_id: "sess-123",
      user_id: "user-456",
      username: "alice",
      resumed: false,
      ...overrides,
    },
  };
}

describe("ReconnectingWS", () => {
  it("connects and sends join with room_id and username", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      username: "alice",
    });
    ws.connect();

    const sock = lastSocket();
    expect(sock.url).toBe("ws://localhost/ws");

    sock.simulateOpen();
    expect(sock.sent).toHaveLength(1);
    const join = JSON.parse(sock.sent[0]);
    expect(join.type).toBe("join");
    expect(join.payload.room_id).toBe("room1");
    expect(join.payload.username).toBe("alice");
    expect(join.payload.session_id).toBeUndefined();

    ws.disconnect();
  });

  it("transitions to connected state on session response", () => {
    const states: ConnectionState[] = [];
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onStateChange: (s) => states.push(s),
    });

    ws.connect();
    expect(states).toEqual(["connecting"]);

    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    expect(states).toEqual(["connecting", "connected"]);
    expect(ws.getState()).toBe("connected");
    expect(ws.getSessionID()).toBe("sess-123");

    ws.disconnect();
  });

  it("calls onSession callback", () => {
    const onSession = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onSession,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ resumed: false }));

    expect(onSession).toHaveBeenCalledWith(
      expect.objectContaining({
        session_id: "sess-123",
        resumed: false,
      }),
    );

    ws.disconnect();
  });

  it("forwards non-session messages to onMessage", () => {
    const onMessage = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onMessage,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());
    lastSocket().simulateMessage({ type: "chat", payload: { content: "hello" } });

    expect(onMessage).toHaveBeenCalledTimes(1);
    expect(onMessage).toHaveBeenCalledWith({
      type: "chat",
      payload: { content: "hello" },
    });

    ws.disconnect();
  });

  it("reconnects with exponential backoff on close", () => {
    const states: ConnectionState[] = [];
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      baseDelay: 100,
      maxDelay: 10_000,
      onStateChange: (s) => states.push(s),
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());
    states.length = 0; // clear initial states

    // Simulate disconnect.
    lastSocket().simulateClose();
    expect(states).toEqual(["reconnecting"]);
    expect(MockWebSocket.instances).toHaveLength(1);

    // Advance past first retry delay (~100ms + jitter).
    vi.advanceTimersByTime(200);
    expect(MockWebSocket.instances).toHaveLength(2);

    // Simulate second connection succeeding.
    lastSocket().simulateOpen();
    // Should include session_id in join.
    const join = JSON.parse(lastSocket().sent[0]);
    expect(join.payload.session_id).toBe("sess-123");

    lastSocket().simulateMessage(sessionEnvelope({ resumed: true }));
    expect(ws.getState()).toBe("connected");

    ws.disconnect();
  });

  it("sends session_id on reconnect join", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      username: "bob",
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ session_id: "my-sess" }));

    // First join should not have session_id.
    const firstJoin = JSON.parse(MockWebSocket.instances[0].sent[0]);
    expect(firstJoin.payload.session_id).toBeUndefined();

    // Disconnect and reconnect.
    lastSocket().simulateClose();
    vi.advanceTimersByTime(1000);

    // Simulate the new socket connecting.
    lastSocket().simulateOpen();

    const secondJoin = JSON.parse(lastSocket().sent[0]);
    expect(secondJoin.payload.session_id).toBe("my-sess");
    expect(secondJoin.payload.room_id).toBe("room1");
    expect(secondJoin.payload.username).toBe("bob");

    ws.disconnect();
  });

  it("stops reconnecting after maxRetries", () => {
    const states: ConnectionState[] = [];
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      maxRetries: 2,
      baseDelay: 50,
      onStateChange: (s) => states.push(s),
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    // Retry 1.
    lastSocket().simulateClose();
    vi.advanceTimersByTime(200);
    lastSocket().simulateClose();

    // Retry 2.
    vi.advanceTimersByTime(500);
    lastSocket().simulateClose();

    // Should stop and become disconnected.
    expect(ws.getState()).toBe("disconnected");
    const instanceCount = MockWebSocket.instances.length;
    vi.advanceTimersByTime(60_000);
    expect(MockWebSocket.instances.length).toBe(instanceCount);

    ws.disconnect();
  });

  it("does not reconnect on policy violation (code 1008)", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      baseDelay: 50,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateClose(1008); // Policy violation

    expect(ws.getState()).toBe("disconnected");
    vi.advanceTimersByTime(10_000);
    expect(MockWebSocket.instances).toHaveLength(1); // No reconnect attempted.

    ws.disconnect();
  });

  it("disconnect stops reconnection attempts", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      baseDelay: 100,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());
    lastSocket().simulateClose();

    // Disconnect before retry fires.
    ws.disconnect();
    expect(ws.getState()).toBe("disconnected");

    vi.advanceTimersByTime(10_000);
    // No new connections should have been made.
    expect(MockWebSocket.instances).toHaveLength(1);
  });

  it("send() does nothing when not connected", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
    });

    // Not connected yet.
    ws.send("chat", { content: "hello" });
    expect(MockWebSocket.instances).toHaveLength(0);

    ws.disconnect();
  });

  it("send() writes to websocket when connected", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    ws.send("chat", { content: "hello" });
    // sent[0] is join, sent[1] is chat
    expect(lastSocket().sent).toHaveLength(2);
    const msg = JSON.parse(lastSocket().sent[1]);
    expect(msg).toEqual({ type: "chat", payload: { content: "hello" } });

    ws.disconnect();
  });

  it("unpacks backfill envelope into individual onMessage calls", () => {
    const onMessage = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onMessage,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ resumed: true }));

    // Simulate backfill envelope with new payload format.
    lastSocket().simulateMessage({
      type: "backfill",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            user_id: "user-1",
            username: "bob",
            content: "hello",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
          {
            id: "msg-2",
            room_id: "room1",
            username: "system",
            content: "alice left the room",
            type: "system",
            created_at: "2026-01-01T00:00:01Z",
          },
          {
            id: "msg-3",
            room_id: "room1",
            user_id: "user-1",
            username: "bob",
            content: "anyone here?",
            type: "chat",
            created_at: "2026-01-01T00:00:02Z",
          },
        ],
        has_gap: false,
      },
    });

    expect(onMessage).toHaveBeenCalledTimes(3);
    expect(onMessage).toHaveBeenNthCalledWith(1, {
      type: "chat",
      payload: expect.objectContaining({ id: "msg-1", content: "hello" }),
    });
    expect(onMessage).toHaveBeenNthCalledWith(2, {
      type: "system",
      payload: expect.objectContaining({
        id: "msg-2",
        content: "alice left the room",
      }),
    });
    expect(onMessage).toHaveBeenNthCalledWith(3, {
      type: "chat",
      payload: expect.objectContaining({
        id: "msg-3",
        content: "anyone here?",
      }),
    });

    ws.disconnect();
  });

  it("does not call onMessage for empty backfill", () => {
    const onMessage = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onMessage,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ resumed: true }));

    lastSocket().simulateMessage({
      type: "backfill",
      payload: {
        messages: [],
        has_gap: false,
      },
    });

    expect(onMessage).not.toHaveBeenCalled();

    ws.disconnect();
  });

  it("calls onBackfillGap when has_gap is true", () => {
    const onMessage = vi.fn();
    const onBackfillGap = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onMessage,
      onBackfillGap,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ resumed: true }));

    lastSocket().simulateMessage({
      type: "backfill",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            user_id: "user-1",
            username: "bob",
            content: "recent message",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        has_gap: true,
      },
    });

    expect(onBackfillGap).toHaveBeenCalledTimes(1);
    expect(onMessage).toHaveBeenCalledTimes(1);

    ws.disconnect();
  });

  it("does not call onBackfillGap when has_gap is false", () => {
    const onBackfillGap = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onBackfillGap,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ resumed: true }));

    lastSocket().simulateMessage({
      type: "backfill",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            content: "hello",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        has_gap: false,
      },
    });

    expect(onBackfillGap).not.toHaveBeenCalled();

    ws.disconnect();
  });

  it("deduplicates backfill messages already seen", () => {
    const onMessage = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onMessage,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    // Receive a live message.
    lastSocket().simulateMessage({
      type: "chat",
      payload: { id: "msg-1", content: "live message" },
    });
    expect(onMessage).toHaveBeenCalledTimes(1);

    // Simulate disconnect and reconnect.
    lastSocket().simulateClose();
    vi.advanceTimersByTime(1000);
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ resumed: true }));

    // Backfill includes the same msg-1 plus a new msg-2.
    lastSocket().simulateMessage({
      type: "backfill",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            content: "live message",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
          {
            id: "msg-2",
            room_id: "room1",
            content: "missed message",
            type: "chat",
            created_at: "2026-01-01T00:00:01Z",
          },
        ],
        has_gap: false,
      },
    });

    // msg-1 should be skipped (already seen), only msg-2 delivered.
    expect(onMessage).toHaveBeenCalledTimes(2); // 1 live + 1 backfill
    expect(onMessage).toHaveBeenLastCalledWith({
      type: "chat",
      payload: expect.objectContaining({ id: "msg-2", content: "missed message" }),
    });

    ws.disconnect();
  });

  it("fetchHistory sends history_fetch envelope", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    ws.fetchHistory("msg-50", 30);
    // sent[0] is join, sent[1] is history_fetch
    expect(lastSocket().sent).toHaveLength(2);
    const fetch = JSON.parse(lastSocket().sent[1]);
    expect(fetch.type).toBe("history_fetch");
    expect(fetch.payload.before_id).toBe("msg-50");
    expect(fetch.payload.limit).toBe(30);

    ws.disconnect();
  });

  it("fetchHistory uses default limit of 50", () => {
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    ws.fetchHistory("msg-50");
    const fetch = JSON.parse(lastSocket().sent[1]);
    expect(fetch.payload.limit).toBe(50);

    ws.disconnect();
  });

  it("dispatches history_batch to onHistoryBatch callback", () => {
    const onHistoryBatch = vi.fn();
    const onMessage = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onHistoryBatch,
      onMessage,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    lastSocket().simulateMessage({
      type: "history_batch",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            content: "old message",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
          {
            id: "msg-2",
            room_id: "room1",
            content: "older message",
            type: "chat",
            created_at: "2026-01-01T00:00:01Z",
          },
        ],
        has_more: true,
      },
    });

    expect(onHistoryBatch).toHaveBeenCalledTimes(1);
    expect(onHistoryBatch).toHaveBeenCalledWith(
      [
        expect.objectContaining({ id: "msg-1", content: "old message" }),
        expect.objectContaining({ id: "msg-2", content: "older message" }),
      ],
      true,
    );
    // history_batch should NOT trigger onMessage
    expect(onMessage).not.toHaveBeenCalled();

    ws.disconnect();
  });

  it("history_batch messages are tracked for deduplication", () => {
    const onMessage = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onHistoryBatch: () => {},
      onMessage,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    // Receive history_batch with msg-1
    lastSocket().simulateMessage({
      type: "history_batch",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            content: "old",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        has_more: false,
      },
    });

    // Now receive msg-1 again as a backfill — should be deduplicated
    lastSocket().simulateMessage({
      type: "backfill",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            content: "old",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        has_gap: false,
      },
    });

    expect(onMessage).not.toHaveBeenCalled();

    ws.disconnect();
  });

  it("history messages are tracked for deduplication", () => {
    const onMessage = vi.fn();
    const ws = new ReconnectingWS({
      url: "ws://localhost/ws",
      roomID: "room1",
      onMessage,
    });

    ws.connect();
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope());

    // Receive history with msg-1.
    lastSocket().simulateMessage({
      type: "history",
      payload: [
        {
          id: "msg-1",
          room_id: "room1",
          content: "from history",
          type: "chat",
          created_at: "2026-01-01T00:00:00Z",
        },
      ],
    });
    expect(onMessage).toHaveBeenCalledTimes(1);

    // Simulate disconnect and reconnect.
    lastSocket().simulateClose();
    vi.advanceTimersByTime(1000);
    lastSocket().simulateOpen();
    lastSocket().simulateMessage(sessionEnvelope({ resumed: true }));

    // Backfill includes the same msg-1.
    lastSocket().simulateMessage({
      type: "backfill",
      payload: {
        messages: [
          {
            id: "msg-1",
            room_id: "room1",
            content: "from history",
            type: "chat",
            created_at: "2026-01-01T00:00:00Z",
          },
        ],
        has_gap: false,
      },
    });

    // msg-1 should be deduplicated — no additional call.
    expect(onMessage).toHaveBeenCalledTimes(1);

    ws.disconnect();
  });
});

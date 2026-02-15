import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useWebSocket } from "./use-websocket";

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

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close() {
    this.readyState = MockWebSocket.CLOSED;
  }

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

function lastSocket(): MockWebSocket {
  return MockWebSocket.instances[MockWebSocket.instances.length - 1];
}

const sessionEnvelope = {
  type: "session",
  payload: {
    session_id: "sess-abc",
    user_id: "user-123",
    username: "alice",
    resumed: false,
  },
};

beforeEach(() => {
  MockWebSocket.instances = [];
  vi.stubGlobal("WebSocket", MockWebSocket);
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("useWebSocket", () => {
  it("starts disconnected then transitions to connected", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://localhost/ws", roomID: "room1" }),
    );

    // Initially connecting.
    expect(result.current.state).toBe("connecting");

    // Simulate server accepting.
    act(() => {
      lastSocket().simulateOpen();
      lastSocket().simulateMessage(sessionEnvelope);
    });

    expect(result.current.state).toBe("connected");
    expect(result.current.session).toEqual(sessionEnvelope.payload);
  });

  it("send() forwards messages when connected", () => {
    const { result } = renderHook(() =>
      useWebSocket({ url: "ws://localhost/ws", roomID: "room1" }),
    );

    act(() => {
      lastSocket().simulateOpen();
      lastSocket().simulateMessage(sessionEnvelope);
    });

    act(() => {
      result.current.send("chat", { content: "hello" });
    });

    expect(lastSocket().sent).toHaveLength(2); // join + chat
    const msg = JSON.parse(lastSocket().sent[1]);
    expect(msg.type).toBe("chat");
  });

  it("calls onMessage for non-session envelopes", () => {
    const onMessage = vi.fn();
    const { result } = renderHook(() =>
      useWebSocket({
        url: "ws://localhost/ws",
        roomID: "room1",
        onMessage,
      }),
    );

    act(() => {
      lastSocket().simulateOpen();
      lastSocket().simulateMessage(sessionEnvelope);
      lastSocket().simulateMessage({ type: "chat", payload: { content: "hi" } });
    });

    expect(onMessage).toHaveBeenCalledTimes(1);
    expect(result.current.state).toBe("connected");
  });

  it("transitions to reconnecting on close", () => {
    const { result } = renderHook(() =>
      useWebSocket({
        url: "ws://localhost/ws",
        roomID: "room1",
      }),
    );

    act(() => {
      lastSocket().simulateOpen();
      lastSocket().simulateMessage(sessionEnvelope);
    });
    expect(result.current.state).toBe("connected");

    act(() => {
      lastSocket().simulateClose();
    });
    expect(result.current.state).toBe("reconnecting");
  });

  it("disconnects on unmount", () => {
    const { unmount } = renderHook(() =>
      useWebSocket({ url: "ws://localhost/ws", roomID: "room1" }),
    );

    act(() => {
      lastSocket().simulateOpen();
      lastSocket().simulateMessage(sessionEnvelope);
    });

    const sock = lastSocket();
    unmount();
    expect(sock.readyState).toBe(MockWebSocket.CLOSED);
  });
});

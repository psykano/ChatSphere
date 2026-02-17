import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, beforeEach, vi } from "vitest";
import App from "./App";

function mockFetchRooms(rooms: unknown[]) {
  vi.spyOn(globalThis, "fetch").mockResolvedValue({
    ok: true,
    json: async () => rooms,
  } as Response);
}

// Stub WebSocket for tests that navigate to ChatLayout
class MockWebSocket {
  static OPEN = 1;
  readyState = MockWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((e: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  send = vi.fn();
  close = vi.fn();
}

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal("WebSocket", MockWebSocket);
  });

  it("renders the heading", async () => {
    mockFetchRooms([]);
    render(<App />);
    expect(
      screen.getByRole("heading", { name: /chatsphere/i })
    ).toBeInTheDocument();
  });

  it("renders the description", async () => {
    mockFetchRooms([]);
    render(<App />);
    expect(
      screen.getByText(/real-time anonymous chat rooms/i)
    ).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    vi.spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));
    render(<App />);
    expect(screen.getByText(/loading rooms/i)).toBeInTheDocument();
  });

  it("shows empty state when no rooms", async () => {
    mockFetchRooms([]);
    render(<App />);
    expect(
      await screen.findByText(/no public rooms yet/i)
    ).toBeInTheDocument();
  });

  it("renders room cards when rooms exist", async () => {
    mockFetchRooms([
      {
        id: "r1",
        name: "General",
        description: "Main chat",
        capacity: 50,
        active_users: 5,
        created_at: "2026-01-01T00:00:00Z",
      },
      {
        id: "r2",
        name: "Gaming",
        capacity: 20,
        active_users: 0,
        created_at: "2026-01-01T00:00:00Z",
      },
    ]);
    render(<App />);
    expect(await screen.findByText("General")).toBeInTheDocument();
    expect(screen.getByText("Gaming")).toBeInTheDocument();
    expect(screen.getByText("Main chat")).toBeInTheDocument();
  });

  it("shows error state on fetch failure", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);
    render(<App />);
    expect(
      await screen.findByText(/failed to fetch rooms: 500/i)
    ).toBeInTheDocument();
  });

  it("shows room code dialog after creating a private room", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      if (init?.method === "POST" && url.endsWith("/rooms")) {
        return {
          ok: true,
          json: async () => ({
            id: "r1",
            name: "Secret Room",
            public: false,
            code: "XYZ789",
            capacity: 50,
            active_users: 0,
            created_at: "2026-01-01T00:00:00Z",
          }),
        } as Response;
      }
      return {
        ok: true,
        json: async () => [],
      } as Response;
    });

    render(<App />);
    await screen.findByText(/no public rooms yet/i);

    await user.type(screen.getByLabelText("Room name"), "Secret Room");
    await user.click(screen.getByLabelText("Public room"));
    await user.click(screen.getByRole("button", { name: /create room/i }));

    expect(
      await screen.findByRole("dialog", { name: /private room created/i })
    ).toBeInTheDocument();
    expect(screen.getByText("XYZ789")).toBeInTheDocument();
  });

  it("navigates to chat layout after creating a public room", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      if (init?.method === "POST") {
        return {
          ok: true,
          json: async () => ({
            id: "r1",
            name: "Public Room",
            public: true,
            capacity: 50,
            active_users: 0,
            created_at: "2026-01-01T00:00:00Z",
          }),
        } as Response;
      }
      return {
        ok: true,
        json: async () => [],
      } as Response;
    });

    render(<App />);
    await screen.findByText(/no public rooms yet/i);

    await user.type(screen.getByLabelText("Room name"), "Public Room");
    await user.click(screen.getByRole("button", { name: /create room/i }));

    // Should navigate to chat layout showing the room name in header
    expect(await screen.findByText("# Public Room")).toBeInTheDocument();
  });

  it("navigates to chat layout after closing room code dialog", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      if (init?.method === "POST") {
        return {
          ok: true,
          json: async () => ({
            id: "r1",
            name: "Secret Room",
            public: false,
            code: "ABC123",
            capacity: 50,
            active_users: 0,
            created_at: "2026-01-01T00:00:00Z",
          }),
        } as Response;
      }
      return {
        ok: true,
        json: async () => [],
      } as Response;
    });

    render(<App />);
    await screen.findByText(/no public rooms yet/i);

    await user.type(screen.getByLabelText("Room name"), "Secret Room");
    await user.click(screen.getByLabelText("Public room"));
    await user.click(screen.getByRole("button", { name: /create room/i }));

    await screen.findByRole("dialog", { name: /private room created/i });
    await user.click(screen.getByRole("button", { name: /done/i }));

    // After closing dialog, should navigate to chat layout
    expect(await screen.findByText("# Secret Room")).toBeInTheDocument();
  });

  it("navigates to chat layout when clicking a room card", async () => {
    const user = userEvent.setup();
    mockFetchRooms([
      {
        id: "r1",
        name: "General",
        description: "Main chat",
        capacity: 50,
        active_users: 5,
        created_at: "2026-01-01T00:00:00Z",
      },
    ]);
    render(<App />);
    await screen.findByText("General");
    await user.click(screen.getByRole("button", { name: /general/i }));

    // Should navigate to chat layout
    expect(await screen.findByText("# General")).toBeInTheDocument();
    expect(screen.getByLabelText("Message input")).toBeInTheDocument();
  });
});

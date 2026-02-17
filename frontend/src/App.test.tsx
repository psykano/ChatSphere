import { render, screen } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import App from "./App";

function mockFetchRooms(rooms: unknown[]) {
  vi.spyOn(globalThis, "fetch").mockResolvedValue({
    ok: true,
    json: async () => rooms,
  } as Response);
}

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
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
});

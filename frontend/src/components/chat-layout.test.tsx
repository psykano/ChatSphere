import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { ChatLayout } from "./chat-layout";
import type { Room } from "@/components/room-card";
import { ThemeProvider } from "@/hooks/use-theme";

const mockRoom: Room = {
  id: "room-1",
  name: "General",
  description: "A general chat room",
  capacity: 50,
  public: true,
  active_users: 3,
  created_at: "2026-02-17T12:00:00Z",
};

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

function renderChatLayout(onLeave = vi.fn()) {
  return render(
    <ThemeProvider>
      <ChatLayout room={mockRoom} onLeave={onLeave} />
    </ThemeProvider>,
  );
}

describe("ChatLayout", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.stubGlobal("WebSocket", MockWebSocket);
  });

  it("renders room name in header", () => {
    renderChatLayout();
    expect(screen.getByText("# General")).toBeInTheDocument();
  });

  it("renders the sidebar toggle button", () => {
    renderChatLayout();
    expect(
      screen.getByRole("button", { name: /open sidebar/i }),
    ).toBeInTheDocument();
  });

  it("toggles sidebar open and closed on button click", async () => {
    const user = userEvent.setup();
    renderChatLayout();

    // Initially the toggle says "Open sidebar"
    const toggleBtn = screen.getByRole("button", { name: /open sidebar/i });
    expect(toggleBtn).toBeInTheDocument();

    // Click to open
    await user.click(toggleBtn);
    expect(
      screen.getByRole("button", { name: /close sidebar/i }),
    ).toBeInTheDocument();

    // Click to close
    await user.click(
      screen.getByRole("button", { name: /close sidebar/i }),
    );
    expect(
      screen.getByRole("button", { name: /open sidebar/i }),
    ).toBeInTheDocument();
  });

  it("renders sidebar content (room info and leave button)", () => {
    renderChatLayout();
    // Sidebar should contain the room name and a leave button
    expect(screen.getByRole("button", { name: /leave room/i })).toBeInTheDocument();
  });

  it("renders message input", () => {
    renderChatLayout();
    expect(screen.getByLabelText("Message input")).toBeInTheDocument();
  });

  it("renders chat messages area", () => {
    renderChatLayout();
    expect(screen.getByRole("log", { name: /chat messages/i })).toBeInTheDocument();
  });

  it("renders theme toggle in header", () => {
    renderChatLayout();
    expect(
      screen.getByRole("button", { name: /switch to/i }),
    ).toBeInTheDocument();
  });

  it("closes sidebar when overlay backdrop is clicked", async () => {
    const user = userEvent.setup();
    renderChatLayout();

    // Open sidebar
    await user.click(screen.getByRole("button", { name: /open sidebar/i }));
    expect(
      screen.getByRole("button", { name: /close sidebar/i }),
    ).toBeInTheDocument();

    // The overlay backdrop should be present
    const overlay = document.querySelector("[aria-hidden='true'].fixed");
    expect(overlay).toBeInTheDocument();

    // Click overlay to close
    await user.click(overlay!);
    expect(
      screen.getByRole("button", { name: /open sidebar/i }),
    ).toBeInTheDocument();
  });

  it("shows online count in header", () => {
    renderChatLayout();
    expect(screen.getByText(/0 online/)).toBeInTheDocument();
  });
});

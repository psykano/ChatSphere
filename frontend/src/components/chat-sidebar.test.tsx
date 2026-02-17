import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { ChatSidebar } from "./chat-sidebar";
import type { Room } from "@/components/room-card";

const mockRoom: Room = {
  id: "room-1",
  name: "General",
  description: "A general chat room",
  capacity: 50,
  public: true,
  active_users: 3,
  created_at: "2026-02-17T12:00:00Z",
};

const mockUsers = [
  { user_id: "u1", username: "Alice" },
  { user_id: "u2", username: "Bob" },
];

describe("ChatSidebar", () => {
  it("renders room name", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={[]}
        connectionState="connected"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByText("General")).toBeInTheDocument();
  });

  it("renders room description", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={[]}
        connectionState="connected"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByText("A general chat room")).toBeInTheDocument();
  });

  it("renders online users", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={mockUsers}
        connectionState="connected"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("Bob")).toBeInTheDocument();
  });

  it("shows online user count", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={mockUsers}
        connectionState="connected"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByText(/Online — 2/)).toBeInTheDocument();
  });

  it("shows connected status", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={[]}
        connectionState="connected"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByText("Connected")).toBeInTheDocument();
  });

  it("shows reconnecting status", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={[]}
        connectionState="reconnecting"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByText("Reconnecting...")).toBeInTheDocument();
  });

  it("shows disconnected status", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={[]}
        connectionState="disconnected"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByText("Disconnected")).toBeInTheDocument();
  });

  it("calls onLeave when leave button is clicked", async () => {
    const user = userEvent.setup();
    const onLeave = vi.fn();
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={[]}
        connectionState="connected"
        onLeave={onLeave}
      />,
    );
    await user.click(screen.getByRole("button", { name: /leave room/i }));
    expect(onLeave).toHaveBeenCalledOnce();
  });

  it("renders online users list with accessible label", () => {
    render(
      <ChatSidebar
        room={mockRoom}
        onlineUsers={mockUsers}
        connectionState="connected"
        onLeave={vi.fn()}
      />,
    );
    expect(screen.getByRole("list", { name: "Online users" })).toBeInTheDocument();
  });
});

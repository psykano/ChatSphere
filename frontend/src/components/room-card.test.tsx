import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { RoomCard, type Room } from "./room-card";

const baseRoom: Room = {
  id: "abc123",
  name: "General Chat",
  description: "A place to hang out",
  capacity: 50,
  active_users: 10,
  created_at: "2026-01-01T00:00:00Z",
};

describe("RoomCard", () => {
  it("renders room name", () => {
    render(<RoomCard room={baseRoom} />);
    expect(screen.getByText("General Chat")).toBeInTheDocument();
  });

  it("renders description when present", () => {
    render(<RoomCard room={baseRoom} />);
    expect(screen.getByText("A place to hang out")).toBeInTheDocument();
  });

  it("does not render description when absent", () => {
    const room = { ...baseRoom, description: undefined };
    render(<RoomCard room={room} />);
    expect(screen.queryByText("A place to hang out")).not.toBeInTheDocument();
  });

  it("shows active users and capacity", () => {
    render(<RoomCard room={baseRoom} />);
    expect(screen.getByText("10/50")).toBeInTheDocument();
  });

  it("shows green indicator when users are active", () => {
    const { container } = render(<RoomCard room={baseRoom} />);
    const indicator = container.querySelector("[aria-hidden='true']");
    expect(indicator?.className).toContain("bg-green-500");
  });

  it("shows muted indicator when no users are active", () => {
    const room = { ...baseRoom, active_users: 0 };
    const { container } = render(<RoomCard room={room} />);
    const indicator = container.querySelector("[aria-hidden='true']");
    expect(indicator?.className).toContain("bg-muted-foreground");
  });

  it("calls onClick with room when clicked", async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<RoomCard room={baseRoom} onClick={onClick} />);
    await user.click(screen.getByRole("button"));
    expect(onClick).toHaveBeenCalledWith(baseRoom);
  });

  it("renders as a button element", () => {
    render(<RoomCard room={baseRoom} />);
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it("shows creator when creator_id is present", () => {
    const room = { ...baseRoom, creator_id: "user42" };
    render(<RoomCard room={room} />);
    expect(screen.getByText("by user42")).toBeInTheDocument();
  });

  it("does not show creator when creator_id is absent", () => {
    render(<RoomCard room={baseRoom} />);
    expect(screen.queryByText(/^by /)).not.toBeInTheDocument();
  });

  it("shows last message content", () => {
    const room: Room = {
      ...baseRoom,
      last_message: {
        content: "Hello everyone!",
        username: "Alice",
        created_at: "2026-01-01T00:30:00Z",
      },
    };
    render(<RoomCard room={room} />);
    expect(screen.getByText("Hello everyone!")).toBeInTheDocument();
    expect(screen.getByText(/Alice:/)).toBeInTheDocument();
  });

  it("shows last message without username", () => {
    const room: Room = {
      ...baseRoom,
      last_message: {
        content: "System message",
        created_at: "2026-01-01T00:30:00Z",
      },
    };
    render(<RoomCard room={room} />);
    expect(screen.getByText("System message")).toBeInTheDocument();
    expect(screen.queryByText(/:/)).not.toBeInTheDocument();
  });

  it("does not show last message section when absent", () => {
    render(<RoomCard room={baseRoom} />);
    expect(screen.queryByText(/ago$/)).not.toBeInTheDocument();
  });

  describe("formatTimeAgo", () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it("shows 'just now' for recent messages", () => {
      vi.setSystemTime(new Date("2026-01-01T00:00:30Z"));
      const room: Room = {
        ...baseRoom,
        last_message: {
          content: "hi",
          created_at: "2026-01-01T00:00:00Z",
        },
      };
      render(<RoomCard room={room} />);
      expect(screen.getByText("just now")).toBeInTheDocument();
    });

    it("shows minutes ago", () => {
      vi.setSystemTime(new Date("2026-01-01T00:05:00Z"));
      const room: Room = {
        ...baseRoom,
        last_message: {
          content: "hi",
          created_at: "2026-01-01T00:00:00Z",
        },
      };
      render(<RoomCard room={room} />);
      expect(screen.getByText("5m ago")).toBeInTheDocument();
    });

    it("shows hours ago", () => {
      vi.setSystemTime(new Date("2026-01-01T03:00:00Z"));
      const room: Room = {
        ...baseRoom,
        last_message: {
          content: "hi",
          created_at: "2026-01-01T00:00:00Z",
        },
      };
      render(<RoomCard room={room} />);
      expect(screen.getByText("3h ago")).toBeInTheDocument();
    });

    it("shows days ago", () => {
      vi.setSystemTime(new Date("2026-01-03T00:00:00Z"));
      const room: Room = {
        ...baseRoom,
        last_message: {
          content: "hi",
          created_at: "2026-01-01T00:00:00Z",
        },
      };
      render(<RoomCard room={room} />);
      expect(screen.getByText("2d ago")).toBeInTheDocument();
    });
  });
});

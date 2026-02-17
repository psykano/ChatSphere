import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
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
});

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { CreateRoomForm } from "./create-room-form";

describe("CreateRoomForm", () => {
  it("renders all form fields", () => {
    render(<CreateRoomForm onSubmit={vi.fn()} />);
    expect(screen.getByLabelText("Room name")).toBeInTheDocument();
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
    expect(screen.getByLabelText("Capacity")).toBeInTheDocument();
    expect(screen.getByLabelText("Public room")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create room/i })).toBeInTheDocument();
  });

  it("disables submit when name is empty", () => {
    render(<CreateRoomForm onSubmit={vi.fn()} />);
    expect(screen.getByRole("button", { name: /create room/i })).toBeDisabled();
  });

  it("enables submit when name is provided", async () => {
    const user = userEvent.setup();
    render(<CreateRoomForm onSubmit={vi.fn()} />);
    await user.type(screen.getByLabelText("Room name"), "My Room");
    expect(screen.getByRole("button", { name: /create room/i })).toBeEnabled();
  });

  it("calls onSubmit with form data", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<CreateRoomForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Room name"), "My Room");
    await user.type(screen.getByLabelText(/description/i), "A cool room");
    await user.clear(screen.getByLabelText("Capacity"));
    await user.type(screen.getByLabelText("Capacity"), "25");
    await user.click(screen.getByRole("button", { name: /create room/i }));

    expect(onSubmit).toHaveBeenCalledWith({
      name: "My Room",
      description: "A cool room",
      capacity: 25,
      public: true,
    });
  });

  it("trims name and description on submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<CreateRoomForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Room name"), "  My Room  ");
    await user.type(screen.getByLabelText(/description/i), "  A room  ");
    await user.click(screen.getByRole("button", { name: /create room/i }));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ name: "My Room", description: "A room" })
    );
  });

  it("toggles public/private visibility", async () => {
    const user = userEvent.setup();
    render(<CreateRoomForm onSubmit={vi.fn()} />);

    const toggle = screen.getByLabelText("Public room");
    expect(toggle).toHaveAttribute("aria-checked", "true");
    expect(screen.getByText("Public")).toBeInTheDocument();

    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-checked", "false");
    expect(screen.getByText("Private")).toBeInTheDocument();
  });

  it("submits with private visibility when toggled", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<CreateRoomForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Room name"), "Secret Room");
    await user.click(screen.getByLabelText("Public room"));
    await user.click(screen.getByRole("button", { name: /create room/i }));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ public: false })
    );
  });

  it("disables submit when capacity is below 2", async () => {
    const user = userEvent.setup();
    render(<CreateRoomForm onSubmit={vi.fn()} />);
    await user.type(screen.getByLabelText("Room name"), "Room");
    await user.clear(screen.getByLabelText("Capacity"));
    await user.type(screen.getByLabelText("Capacity"), "1");
    expect(screen.getByRole("button", { name: /create room/i })).toBeDisabled();
  });

  it("disables submit when capacity is above 100", async () => {
    const user = userEvent.setup();
    render(<CreateRoomForm onSubmit={vi.fn()} />);
    await user.type(screen.getByLabelText("Room name"), "Room");
    await user.clear(screen.getByLabelText("Capacity"));
    await user.type(screen.getByLabelText("Capacity"), "101");
    expect(screen.getByRole("button", { name: /create room/i })).toBeDisabled();
  });

  it("shows loading state", () => {
    render(<CreateRoomForm onSubmit={vi.fn()} loading />);
    expect(screen.getByRole("button", { name: /creating/i })).toBeDisabled();
  });

  it("shows error message", () => {
    render(<CreateRoomForm onSubmit={vi.fn()} error="Rate limit exceeded" />);
    expect(screen.getByRole("alert")).toHaveTextContent("Rate limit exceeded");
  });

  it("does not show error when error is null", () => {
    render(<CreateRoomForm onSubmit={vi.fn()} error={null} />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("does not call onSubmit when loading", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<CreateRoomForm onSubmit={onSubmit} loading />);
    await user.type(screen.getByLabelText("Room name"), "Room");
    // Button is disabled so click won't fire submit
    await user.click(screen.getByRole("button", { name: /creating/i }));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("defaults capacity to 50", () => {
    render(<CreateRoomForm onSubmit={vi.fn()} />);
    expect(screen.getByLabelText("Capacity")).toHaveValue(50);
  });

  it("truncates name to 100 characters", async () => {
    const user = userEvent.setup();
    render(<CreateRoomForm onSubmit={vi.fn()} />);
    const longName = "a".repeat(110);
    await user.type(screen.getByLabelText("Room name"), longName);
    expect(screen.getByLabelText("Room name")).toHaveValue("a".repeat(100));
  });
});

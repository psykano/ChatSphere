import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { EnterCodeBar } from "./enter-code-bar";

describe("EnterCodeBar", () => {
  it("renders input and join button", () => {
    render(<EnterCodeBar onJoin={vi.fn()} />);
    expect(screen.getByLabelText("Room code")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /join/i })).toBeInTheDocument();
  });

  it("disables join button when code is empty", () => {
    render(<EnterCodeBar onJoin={vi.fn()} />);
    expect(screen.getByRole("button", { name: /join/i })).toBeDisabled();
  });

  it("disables join button when code is too short", async () => {
    const user = userEvent.setup();
    render(<EnterCodeBar onJoin={vi.fn()} />);
    await user.type(screen.getByLabelText("Room code"), "ABC");
    expect(screen.getByRole("button", { name: /join/i })).toBeDisabled();
  });

  it("enables join button when code is 6 alphanumeric characters", async () => {
    const user = userEvent.setup();
    render(<EnterCodeBar onJoin={vi.fn()} />);
    await user.type(screen.getByLabelText("Room code"), "ABC123");
    expect(screen.getByRole("button", { name: /join/i })).toBeEnabled();
  });

  it("calls onJoin with uppercase code on submit", async () => {
    const user = userEvent.setup();
    const onJoin = vi.fn();
    render(<EnterCodeBar onJoin={onJoin} />);
    await user.type(screen.getByLabelText("Room code"), "abc123");
    await user.click(screen.getByRole("button", { name: /join/i }));
    expect(onJoin).toHaveBeenCalledWith("ABC123");
  });

  it("calls onJoin on enter key", async () => {
    const user = userEvent.setup();
    const onJoin = vi.fn();
    render(<EnterCodeBar onJoin={onJoin} />);
    await user.type(screen.getByLabelText("Room code"), "ABC123{enter}");
    expect(onJoin).toHaveBeenCalledWith("ABC123");
  });

  it("truncates input to 6 characters", async () => {
    const user = userEvent.setup();
    render(<EnterCodeBar onJoin={vi.fn()} />);
    await user.type(screen.getByLabelText("Room code"), "ABCDEFGH");
    expect(screen.getByLabelText("Room code")).toHaveValue("ABCDEF");
  });

  it("shows loading state", () => {
    render(<EnterCodeBar onJoin={vi.fn()} loading />);
    expect(screen.getByRole("button", { name: /joining/i })).toBeDisabled();
  });

  it("shows error message", () => {
    render(<EnterCodeBar onJoin={vi.fn()} error="Room not found" />);
    expect(screen.getByRole("alert")).toHaveTextContent("Room not found");
  });

  it("does not show error when error is null", () => {
    render(<EnterCodeBar onJoin={vi.fn()} error={null} />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("does not call onJoin when loading", async () => {
    const user = userEvent.setup();
    const onJoin = vi.fn();
    render(<EnterCodeBar onJoin={onJoin} loading />);
    await user.type(screen.getByLabelText("Room code"), "ABC123{enter}");
    expect(onJoin).not.toHaveBeenCalled();
  });
});

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { UsernameInput } from "./username-input";

describe("UsernameInput", () => {
  it("renders the input and join button", () => {
    render(<UsernameInput onSubmit={vi.fn()} />);
    expect(screen.getByLabelText("Username")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Join" })).toBeInTheDocument();
  });

  it("disables the join button when input is empty", () => {
    render(<UsernameInput onSubmit={vi.fn()} />);
    expect(screen.getByRole("button", { name: "Join" })).toBeDisabled();
  });

  it("enables the join button when input has text", async () => {
    const user = userEvent.setup();
    render(<UsernameInput onSubmit={vi.fn()} />);

    await user.type(screen.getByLabelText("Username"), "Alice");
    expect(screen.getByRole("button", { name: "Join" })).toBeEnabled();
  });

  it("calls onSubmit with trimmed username on form submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<UsernameInput onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Username"), "  Alice  ");
    await user.click(screen.getByRole("button", { name: "Join" }));

    expect(onSubmit).toHaveBeenCalledWith("Alice");
  });

  it("calls onSubmit when Enter is pressed", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<UsernameInput onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Username"), "Bob{Enter}");
    expect(onSubmit).toHaveBeenCalledWith("Bob");
  });

  it("does not call onSubmit when input is whitespace only", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<UsernameInput onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Username"), "   {Enter}");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("has a maxLength of 32", () => {
    render(<UsernameInput onSubmit={vi.fn()} />);
    expect(screen.getByLabelText("Username")).toHaveAttribute("maxLength", "32");
  });

  it("has the set username form label", () => {
    render(<UsernameInput onSubmit={vi.fn()} />);
    expect(screen.getByRole("form", { name: "Set username" })).toBeInTheDocument();
  });
});

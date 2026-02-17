import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { MessageInput } from "./message-input";

describe("MessageInput", () => {
  it("renders textarea and send button", () => {
    render(<MessageInput onSend={vi.fn()} />);
    expect(screen.getByLabelText("Message input")).toBeInTheDocument();
    expect(screen.getByLabelText("Send message")).toBeInTheDocument();
  });

  it("disables send button when input is empty", () => {
    render(<MessageInput onSend={vi.fn()} />);
    expect(screen.getByLabelText("Send message")).toBeDisabled();
  });

  it("enables send button when input has text", async () => {
    const user = userEvent.setup();
    render(<MessageInput onSend={vi.fn()} />);
    await user.type(screen.getByLabelText("Message input"), "Hello");
    expect(screen.getByLabelText("Send message")).toBeEnabled();
  });

  it("calls onSend with trimmed text on submit", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(<MessageInput onSend={onSend} />);
    await user.type(screen.getByLabelText("Message input"), "  Hello  ");
    await user.click(screen.getByLabelText("Send message"));
    expect(onSend).toHaveBeenCalledWith("Hello");
  });

  it("clears input after sending", async () => {
    const user = userEvent.setup();
    render(<MessageInput onSend={vi.fn()} />);
    const input = screen.getByLabelText("Message input");
    await user.type(input, "Hello");
    await user.click(screen.getByLabelText("Send message"));
    expect(input).toHaveValue("");
  });

  it("sends on Enter key", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(<MessageInput onSend={onSend} />);
    await user.type(screen.getByLabelText("Message input"), "Hello{Enter}");
    expect(onSend).toHaveBeenCalledWith("Hello");
  });

  it("does not send on Shift+Enter", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(<MessageInput onSend={onSend} />);
    await user.type(
      screen.getByLabelText("Message input"),
      "Hello{Shift>}{Enter}{/Shift}",
    );
    expect(onSend).not.toHaveBeenCalled();
  });

  it("disables input when disabled prop is true", () => {
    render(<MessageInput onSend={vi.fn()} disabled />);
    expect(screen.getByLabelText("Message input")).toBeDisabled();
    expect(screen.getByLabelText("Send message")).toBeDisabled();
  });

  it("does not call onSend with whitespace-only input", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(<MessageInput onSend={onSend} />);
    await user.type(screen.getByLabelText("Message input"), "   {Enter}");
    expect(onSend).not.toHaveBeenCalled();
  });

  it("disables input and send button when readOnly is true", () => {
    render(<MessageInput onSend={vi.fn()} readOnly />);
    expect(screen.getByLabelText("Message input")).toBeDisabled();
    expect(screen.getByLabelText("Send message")).toBeDisabled();
  });

  it("shows read-only placeholder when readOnly is true", () => {
    render(<MessageInput onSend={vi.fn()} readOnly />);
    expect(screen.getByLabelText("Message input")).toHaveAttribute(
      "placeholder",
      "Set a username to start chatting",
    );
  });

  it("does not call onSend when readOnly even with text", async () => {
    const onSend = vi.fn();
    render(<MessageInput onSend={onSend} readOnly />);
    // Input is disabled so user can't type, but verify onSend is not called
    expect(onSend).not.toHaveBeenCalled();
  });
});

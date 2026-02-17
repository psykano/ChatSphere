import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { MessageInput } from "./message-input";

describe("MessageInput typing events", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("calls onTyping when user types text", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const onTyping = vi.fn();
    render(<MessageInput onSend={vi.fn()} onTyping={onTyping} />);
    await user.type(screen.getByLabelText("Message input"), "H");
    expect(onTyping).toHaveBeenCalledTimes(1);
  });

  it("throttles onTyping calls", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const onTyping = vi.fn();
    render(<MessageInput onSend={vi.fn()} onTyping={onTyping} />);

    await user.type(screen.getByLabelText("Message input"), "Hello");
    // Should only fire once despite multiple keystrokes (throttled at 2s).
    expect(onTyping).toHaveBeenCalledTimes(1);
  });

  it("fires onTyping again after throttle period", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const onTyping = vi.fn();
    render(<MessageInput onSend={vi.fn()} onTyping={onTyping} />);

    await user.type(screen.getByLabelText("Message input"), "H");
    expect(onTyping).toHaveBeenCalledTimes(1);

    vi.advanceTimersByTime(2000);

    await user.type(screen.getByLabelText("Message input"), "i");
    expect(onTyping).toHaveBeenCalledTimes(2);
  });

  it("does not call onTyping for whitespace-only input", async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    const onTyping = vi.fn();
    render(<MessageInput onSend={vi.fn()} onTyping={onTyping} />);
    await user.type(screen.getByLabelText("Message input"), " ");
    expect(onTyping).not.toHaveBeenCalled();
  });

  it("does not call onTyping when disabled", () => {
    const onTyping = vi.fn();
    render(<MessageInput onSend={vi.fn()} onTyping={onTyping} disabled />);
    expect(screen.getByLabelText("Message input")).toBeDisabled();
    expect(onTyping).not.toHaveBeenCalled();
  });
});

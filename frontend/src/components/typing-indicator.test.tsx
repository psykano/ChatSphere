import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { TypingIndicator } from "./typing-indicator";

describe("TypingIndicator", () => {
  it("renders nothing when no users are typing", () => {
    const { container } = render(<TypingIndicator typingUsers={new Map()} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("shows single user typing", () => {
    const users = new Map([["user-1", "Alice"]]);
    render(<TypingIndicator typingUsers={users} />);
    expect(screen.getByText("Alice is typing...")).toBeInTheDocument();
  });

  it("shows two users typing", () => {
    const users = new Map([
      ["user-1", "Alice"],
      ["user-2", "Bob"],
    ]);
    render(<TypingIndicator typingUsers={users} />);
    expect(screen.getByText("Alice and Bob are typing...")).toBeInTheDocument();
  });

  it("shows generic message for three or more users", () => {
    const users = new Map([
      ["user-1", "Alice"],
      ["user-2", "Bob"],
      ["user-3", "Charlie"],
    ]);
    render(<TypingIndicator typingUsers={users} />);
    expect(screen.getByText("Several people are typing...")).toBeInTheDocument();
  });

  it("has accessible aria attributes", () => {
    const users = new Map([["user-1", "Alice"]]);
    render(<TypingIndicator typingUsers={users} />);
    const indicator = screen.getByLabelText("Typing indicator");
    expect(indicator).toHaveAttribute("aria-live", "polite");
  });
});

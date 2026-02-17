import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MessageBubble } from "./message-bubble";
import type { ChatMessage } from "@/hooks/use-chat";

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
  return {
    id: "msg-1",
    room_id: "room-1",
    user_id: "user-1",
    username: "Alice",
    content: "Hello world",
    type: "message",
    created_at: "2026-02-17T12:00:00Z",
    ...overrides,
  };
}

describe("MessageBubble", () => {
  it("renders message content", () => {
    render(<MessageBubble message={makeMessage()} isOwn={false} />);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("shows username for other users' messages", () => {
    render(<MessageBubble message={makeMessage()} isOwn={false} />);
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("hides username for own messages", () => {
    render(<MessageBubble message={makeMessage()} isOwn={true} />);
    expect(screen.queryByText("Alice")).not.toBeInTheDocument();
  });

  it("renders system message as italic centered text", () => {
    render(
      <MessageBubble
        message={makeMessage({ type: "system", content: "Room expires in 5 minutes" })}
        isOwn={false}
      />,
    );
    const el = screen.getByText("Room expires in 5 minutes");
    expect(el).toBeInTheDocument();
    expect(el.tagName).toBe("SPAN");
  });

  it("renders join message as system message", () => {
    render(
      <MessageBubble
        message={makeMessage({ type: "join", content: "Alice joined" })}
        isOwn={false}
      />,
    );
    expect(screen.getByText("Alice joined")).toBeInTheDocument();
  });

  it("renders leave message as system message", () => {
    render(
      <MessageBubble
        message={makeMessage({ type: "leave", content: "Alice left" })}
        isOwn={false}
      />,
    );
    expect(screen.getByText("Alice left")).toBeInTheDocument();
  });

  it("shows timestamp", () => {
    render(<MessageBubble message={makeMessage()} isOwn={false} />);
    // The exact format depends on locale, just check something time-like is present
    const timeEl = screen.getByText(/\d{1,2}:\d{2}/);
    expect(timeEl).toBeInTheDocument();
  });
});

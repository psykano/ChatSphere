import { describe, it, expect } from "vitest";
import { isSameUserAsPrevious } from "./message-grouping";
import type { ChatMessage } from "@/hooks/use-chat";

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
  return {
    id: "msg-1",
    room_id: "room-1",
    user_id: "user-1",
    username: "Alice",
    content: "Hello",
    type: "message",
    created_at: "2026-02-17T12:00:00Z",
    ...overrides,
  };
}

describe("isSameUserAsPrevious", () => {
  it("returns false for the first message", () => {
    const messages = [makeMessage()];
    expect(isSameUserAsPrevious(messages, 0)).toBe(false);
  });

  it("returns true when consecutive messages are from the same user", () => {
    const messages = [
      makeMessage({ id: "1", user_id: "user-1" }),
      makeMessage({ id: "2", user_id: "user-1" }),
    ];
    expect(isSameUserAsPrevious(messages, 1)).toBe(true);
  });

  it("returns false when consecutive messages are from different users", () => {
    const messages = [
      makeMessage({ id: "1", user_id: "user-1" }),
      makeMessage({ id: "2", user_id: "user-2", username: "Bob" }),
    ];
    expect(isSameUserAsPrevious(messages, 1)).toBe(false);
  });

  it("returns false when current message is a system message", () => {
    const messages = [
      makeMessage({ id: "1", user_id: "user-1" }),
      makeMessage({ id: "2", user_id: "user-1", type: "system", content: "Room updated" }),
    ];
    expect(isSameUserAsPrevious(messages, 1)).toBe(false);
  });

  it("returns false when previous message is a system message", () => {
    const messages = [
      makeMessage({ id: "1", type: "join", content: "Alice joined" }),
      makeMessage({ id: "2", user_id: "user-1" }),
    ];
    expect(isSameUserAsPrevious(messages, 1)).toBe(false);
  });

  it("returns false when previous message is a leave message", () => {
    const messages = [
      makeMessage({ id: "1", type: "leave", content: "Alice left" }),
      makeMessage({ id: "2", user_id: "user-1" }),
    ];
    expect(isSameUserAsPrevious(messages, 1)).toBe(false);
  });

  it("groups multiple consecutive messages from the same user", () => {
    const messages = [
      makeMessage({ id: "1", user_id: "user-1" }),
      makeMessage({ id: "2", user_id: "user-1" }),
      makeMessage({ id: "3", user_id: "user-1" }),
    ];
    expect(isSameUserAsPrevious(messages, 0)).toBe(false);
    expect(isSameUserAsPrevious(messages, 1)).toBe(true);
    expect(isSameUserAsPrevious(messages, 2)).toBe(true);
  });

  it("breaks grouping when a different user interjects", () => {
    const messages = [
      makeMessage({ id: "1", user_id: "user-1" }),
      makeMessage({ id: "2", user_id: "user-2", username: "Bob" }),
      makeMessage({ id: "3", user_id: "user-1" }),
    ];
    expect(isSameUserAsPrevious(messages, 1)).toBe(false);
    expect(isSameUserAsPrevious(messages, 2)).toBe(false);
  });

  it("breaks grouping when a system message interjects", () => {
    const messages = [
      makeMessage({ id: "1", user_id: "user-1" }),
      makeMessage({ id: "2", type: "system", content: "Room expires soon" }),
      makeMessage({ id: "3", user_id: "user-1" }),
    ];
    expect(isSameUserAsPrevious(messages, 2)).toBe(false);
  });
});

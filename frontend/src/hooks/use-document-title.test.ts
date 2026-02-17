import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDocumentTitle } from "./use-document-title";

describe("useDocumentTitle", () => {
  let hiddenValue = false;

  beforeEach(() => {
    document.title = "ChatSphere";
    hiddenValue = false;
    Object.defineProperty(document, "hidden", {
      get: () => hiddenValue,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("sets title to ChatSphere on mount", () => {
    renderHook(() => useDocumentTitle());
    expect(document.title).toBe("ChatSphere");
  });

  it("does not increment unread when tab is visible", () => {
    const { result } = renderHook(() => useDocumentTitle());

    act(() => {
      result.current.incrementUnread();
    });

    expect(result.current.unreadCount).toBe(0);
    expect(document.title).toBe("ChatSphere");
  });

  it("increments unread count when tab is hidden", () => {
    const { result } = renderHook(() => useDocumentTitle());

    hiddenValue = true;
    document.dispatchEvent(new Event("visibilitychange"));

    act(() => {
      result.current.incrementUnread();
    });

    expect(result.current.unreadCount).toBe(1);
    expect(document.title).toBe("(1) ChatSphere");
  });

  it("accumulates unread count for multiple messages", () => {
    const { result } = renderHook(() => useDocumentTitle());

    hiddenValue = true;
    document.dispatchEvent(new Event("visibilitychange"));

    act(() => {
      result.current.incrementUnread();
      result.current.incrementUnread();
      result.current.incrementUnread();
    });

    expect(result.current.unreadCount).toBe(3);
    expect(document.title).toBe("(3) ChatSphere");
  });

  it("resets unread count when tab becomes visible", () => {
    const { result } = renderHook(() => useDocumentTitle());

    // Go hidden and add messages
    hiddenValue = true;
    document.dispatchEvent(new Event("visibilitychange"));

    act(() => {
      result.current.incrementUnread();
      result.current.incrementUnread();
    });

    expect(result.current.unreadCount).toBe(2);

    // Come back to visible
    act(() => {
      hiddenValue = false;
      document.dispatchEvent(new Event("visibilitychange"));
    });

    expect(result.current.unreadCount).toBe(0);
    expect(document.title).toBe("ChatSphere");
  });

  it("restores base title on unmount", () => {
    const { result, unmount } = renderHook(() => useDocumentTitle());

    hiddenValue = true;
    document.dispatchEvent(new Event("visibilitychange"));

    act(() => {
      result.current.incrementUnread();
    });

    expect(document.title).toBe("(1) ChatSphere");

    unmount();
    expect(document.title).toBe("ChatSphere");
  });
});

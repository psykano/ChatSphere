import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useRooms } from "./use-rooms";

const mockRooms = [
  {
    id: "r1",
    name: "General",
    capacity: 50,
    active_users: 5,
    created_at: "2026-01-01T00:00:00Z",
  },
];

describe("useRooms", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("fetches rooms on mount", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      json: async () => mockRooms,
    } as Response);

    const { result } = renderHook(() => useRooms());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.rooms).toEqual(mockRooms);
    expect(result.current.error).toBeNull();
  });

  it("starts in loading state", () => {
    vi.spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));
    const { result } = renderHook(() => useRooms());
    expect(result.current.loading).toBe(true);
    expect(result.current.rooms).toEqual([]);
  });

  it("sets error on fetch failure", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 500,
    } as Response);

    const { result } = renderHook(() => useRooms());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.error).toBe("Failed to fetch rooms: 500");
    expect(result.current.rooms).toEqual([]);
  });

  it("sets error on network failure", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(
      new Error("Network error")
    );

    const { result } = renderHook(() => useRooms());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    expect(result.current.error).toBe("Network error");
  });

  it("polls for updates", async () => {
    vi.useFakeTimers();
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      json: async () => mockRooms,
    } as Response);

    renderHook(() => useRooms());

    // Flush the initial fetch
    await act(async () => {
      await vi.advanceTimersByTimeAsync(0);
    });

    expect(fetchSpy).toHaveBeenCalledTimes(1);

    // Advance past the poll interval
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });

    expect(fetchSpy).toHaveBeenCalledTimes(2);
    vi.useRealTimers();
  });

  it("clears error on successful refresh after failure", async () => {
    let callCount = 0;
    vi.spyOn(globalThis, "fetch").mockImplementation(async () => {
      callCount++;
      if (callCount === 1) {
        return { ok: false, status: 500 } as Response;
      }
      return {
        ok: true,
        json: async () => mockRooms,
      } as Response;
    });

    const { result } = renderHook(() => useRooms());

    await waitFor(() => {
      expect(result.current.error).toBe("Failed to fetch rooms: 500");
    });

    // Manually trigger refresh
    await act(async () => {
      await result.current.refresh();
    });

    expect(result.current.error).toBeNull();
    expect(result.current.rooms).toEqual(mockRooms);
  });
});

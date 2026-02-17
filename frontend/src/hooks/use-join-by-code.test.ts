import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useJoinByCode } from "./use-join-by-code";

describe("useJoinByCode", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("starts with no loading and no error", () => {
    const { result } = renderHook(() => useJoinByCode());
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("returns room on successful lookup", async () => {
    const mockRoom = { id: "r1", name: "Secret", capacity: 10, active_users: 2, created_at: "2026-01-01T00:00:00Z" };
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      json: async () => mockRoom,
    } as Response);

    const { result } = renderHook(() => useJoinByCode());

    let room: unknown;
    await act(async () => {
      room = await result.current.joinByCode("ABC123");
    });

    expect(room).toEqual(mockRoom);
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/rooms/code/ABC123");
  });

  it("sets error on 404 response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 404,
      json: async () => ({ error: "room not found" }),
    } as Response);

    const { result } = renderHook(() => useJoinByCode());

    let room: unknown;
    await act(async () => {
      room = await result.current.joinByCode("BADCODE");
    });

    expect(room).toBeNull();
    expect(result.current.error).toBe("room not found");
    expect(result.current.loading).toBe(false);
  });

  it("sets error on network failure", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useJoinByCode());

    let room: unknown;
    await act(async () => {
      room = await result.current.joinByCode("ABC123");
    });

    expect(room).toBeNull();
    expect(result.current.error).toBe("Network error");
  });

  it("sets loading during fetch", async () => {
    let resolveFetch: (value: Response) => void;
    vi.spyOn(globalThis, "fetch").mockReturnValue(
      new Promise((resolve) => { resolveFetch = resolve; })
    );

    const { result } = renderHook(() => useJoinByCode());

    let joinPromise: Promise<unknown>;
    act(() => {
      joinPromise = result.current.joinByCode("ABC123");
    });

    expect(result.current.loading).toBe(true);

    await act(async () => {
      resolveFetch!({
        ok: true,
        json: async () => ({ id: "r1", name: "Test", capacity: 10, active_users: 0, created_at: "2026-01-01T00:00:00Z" }),
      } as Response);
      await joinPromise;
    });

    expect(result.current.loading).toBe(false);
  });

  it("clears error on clearError", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("fail"));

    const { result } = renderHook(() => useJoinByCode());

    await act(async () => {
      await result.current.joinByCode("ABC123");
    });

    expect(result.current.error).toBe("fail");

    act(() => {
      result.current.clearError();
    });

    expect(result.current.error).toBeNull();
  });

  it("clears previous error on new request", async () => {
    vi.spyOn(globalThis, "fetch")
      .mockRejectedValueOnce(new Error("fail"))
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: "r1", name: "Test", capacity: 10, active_users: 0, created_at: "2026-01-01T00:00:00Z" }),
      } as Response);

    const { result } = renderHook(() => useJoinByCode());

    await act(async () => {
      await result.current.joinByCode("BAD");
    });
    expect(result.current.error).toBe("fail");

    await act(async () => {
      await result.current.joinByCode("GOOD12");
    });
    expect(result.current.error).toBeNull();
  });

  it("handles non-JSON error response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => { throw new Error("not json"); },
    } as Response);

    const { result } = renderHook(() => useJoinByCode());

    await act(async () => {
      await result.current.joinByCode("ABC123");
    });

    expect(result.current.error).toBe("Room not found (500)");
  });
});

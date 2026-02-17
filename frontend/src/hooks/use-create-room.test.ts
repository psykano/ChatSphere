import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { useCreateRoom } from "./use-create-room";

const validInput = {
  name: "Test Room",
  description: "A test room",
  capacity: 50,
  public: true,
};

const mockRoom = {
  id: "r1",
  name: "Test Room",
  description: "A test room",
  capacity: 50,
  active_users: 0,
  created_at: "2026-01-01T00:00:00Z",
};

describe("useCreateRoom", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("starts with no loading and no error", () => {
    const { result } = renderHook(() => useCreateRoom());
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("returns room on successful creation", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      json: async () => mockRoom,
    } as Response);

    const { result } = renderHook(() => useCreateRoom());

    let room: unknown;
    await act(async () => {
      room = await result.current.createRoom(validInput);
    });

    expect(room).toEqual(mockRoom);
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
    expect(globalThis.fetch).toHaveBeenCalledWith("/api/rooms", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(validInput),
    });
  });

  it("sets error on failure response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: "name is required" }),
    } as Response);

    const { result } = renderHook(() => useCreateRoom());

    let room: unknown;
    await act(async () => {
      room = await result.current.createRoom(validInput);
    });

    expect(room).toBeNull();
    expect(result.current.error).toBe("name is required");
  });

  it("sets error on rate limit response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 429,
      json: async () => ({ error: "rate limit exceeded, max 3 rooms per hour" }),
    } as Response);

    const { result } = renderHook(() => useCreateRoom());

    await act(async () => {
      await result.current.createRoom(validInput);
    });

    expect(result.current.error).toBe("rate limit exceeded, max 3 rooms per hour");
  });

  it("sets error on network failure", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("Network error"));

    const { result } = renderHook(() => useCreateRoom());

    let room: unknown;
    await act(async () => {
      room = await result.current.createRoom(validInput);
    });

    expect(room).toBeNull();
    expect(result.current.error).toBe("Network error");
  });

  it("sets loading during fetch", async () => {
    let resolveFetch: (value: Response) => void;
    vi.spyOn(globalThis, "fetch").mockReturnValue(
      new Promise((resolve) => { resolveFetch = resolve; })
    );

    const { result } = renderHook(() => useCreateRoom());

    let createPromise: Promise<unknown>;
    act(() => {
      createPromise = result.current.createRoom(validInput);
    });

    expect(result.current.loading).toBe(true);

    await act(async () => {
      resolveFetch!({
        ok: true,
        json: async () => mockRoom,
      } as Response);
      await createPromise;
    });

    expect(result.current.loading).toBe(false);
  });

  it("clears error on clearError", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("fail"));

    const { result } = renderHook(() => useCreateRoom());

    await act(async () => {
      await result.current.createRoom(validInput);
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
        json: async () => mockRoom,
      } as Response);

    const { result } = renderHook(() => useCreateRoom());

    await act(async () => {
      await result.current.createRoom(validInput);
    });
    expect(result.current.error).toBe("fail");

    await act(async () => {
      await result.current.createRoom(validInput);
    });
    expect(result.current.error).toBeNull();
  });

  it("handles non-JSON error response", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => { throw new Error("not json"); },
    } as Response);

    const { result } = renderHook(() => useCreateRoom());

    await act(async () => {
      await result.current.createRoom(validInput);
    });

    expect(result.current.error).toBe("Failed to create room (500)");
  });
});

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { ConnectionStatusBanner } from "./connection-status-banner";

describe("ConnectionStatusBanner", () => {
  it("renders nothing when connected", () => {
    const { container } = render(
      <ConnectionStatusBanner connectionState="connected" />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("shows reconnecting message with status role", () => {
    render(<ConnectionStatusBanner connectionState="reconnecting" />);
    const banner = screen.getByRole("status");
    expect(banner).toHaveTextContent("Connection lost. Reconnecting...");
  });

  it("shows connecting message with status role", () => {
    render(<ConnectionStatusBanner connectionState="connecting" />);
    const banner = screen.getByRole("status");
    expect(banner).toHaveTextContent("Connecting...");
  });

  it("shows disconnected message with alert role", () => {
    render(<ConnectionStatusBanner connectionState="disconnected" />);
    const banner = screen.getByRole("alert");
    expect(banner).toHaveTextContent("Disconnected from server");
  });

  it("uses yellow styling for reconnecting state", () => {
    render(<ConnectionStatusBanner connectionState="reconnecting" />);
    const banner = screen.getByRole("status");
    expect(banner.className).toContain("yellow");
  });

  it("uses red styling for disconnected state", () => {
    render(<ConnectionStatusBanner connectionState="disconnected" />);
    const banner = screen.getByRole("alert");
    expect(banner.className).toContain("red");
  });

  it("shows retry button when disconnected and onRetry is provided", () => {
    const onRetry = vi.fn();
    render(<ConnectionStatusBanner connectionState="disconnected" onRetry={onRetry} />);
    expect(screen.getByRole("button", { name: "Retry" })).toBeDefined();
  });

  it("does not show retry button when disconnected and onRetry is not provided", () => {
    render(<ConnectionStatusBanner connectionState="disconnected" />);
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
  });

  it("calls onRetry when retry button is clicked", async () => {
    const onRetry = vi.fn();
    render(<ConnectionStatusBanner connectionState="disconnected" onRetry={onRetry} />);
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("does not show retry button when reconnecting", () => {
    const onRetry = vi.fn();
    render(<ConnectionStatusBanner connectionState="reconnecting" onRetry={onRetry} />);
    expect(screen.queryByRole("button", { name: "Retry" })).toBeNull();
  });
});

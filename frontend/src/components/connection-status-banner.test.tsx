import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
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
});

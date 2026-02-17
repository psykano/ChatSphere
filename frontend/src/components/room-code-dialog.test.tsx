import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { RoomCodeDialog } from "./room-code-dialog";

describe("RoomCodeDialog", () => {
  const defaultProps = {
    roomName: "Secret Room",
    code: "ABC123",
    onClose: vi.fn(),
  };

  it("renders the dialog with room name and code", () => {
    render(<RoomCodeDialog {...defaultProps} />);
    expect(
      screen.getByRole("dialog", { name: /private room created/i })
    ).toBeInTheDocument();
    expect(screen.getByText("Secret Room")).toBeInTheDocument();
    expect(screen.getByText("ABC123")).toBeInTheDocument();
  });

  it("displays the code in a prominent format", () => {
    render(<RoomCodeDialog {...defaultProps} />);
    const codeElement = screen.getByText("ABC123");
    expect(codeElement.tagName).toBe("CODE");
  });

  it("calls onClose when Done button is clicked", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(<RoomCodeDialog {...defaultProps} onClose={onClose} />);
    await user.click(screen.getByRole("button", { name: /done/i }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("copies code to clipboard when Copy is clicked", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      ...navigator,
      clipboard: { writeText },
    });

    render(<RoomCodeDialog {...defaultProps} />);
    await user.click(screen.getByRole("button", { name: /copy/i }));
    expect(writeText).toHaveBeenCalledWith("ABC123");
    vi.unstubAllGlobals();
  });

  it("shows 'Copied!' after clicking Copy", async () => {
    const user = userEvent.setup();
    vi.stubGlobal("navigator", {
      ...navigator,
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    });

    render(<RoomCodeDialog {...defaultProps} />);
    await user.click(screen.getByRole("button", { name: /copy/i }));
    expect(screen.getByText("Copied!")).toBeInTheDocument();
    vi.unstubAllGlobals();
  });

  it("renders Copy and Done buttons", () => {
    render(<RoomCodeDialog {...defaultProps} />);
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /done/i })).toBeInTheDocument();
  });
});

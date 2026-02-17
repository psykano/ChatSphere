import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { EmojiPicker } from "./emoji-picker";

describe("EmojiPicker", () => {
  it("renders the toggle button", () => {
    render(<EmojiPicker onSelect={vi.fn()} />);
    expect(screen.getByLabelText("Open emoji picker")).toBeInTheDocument();
  });

  it("does not show picker dialog initially", () => {
    render(<EmojiPicker onSelect={vi.fn()} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("opens picker on button click", async () => {
    const user = userEvent.setup();
    render(<EmojiPicker onSelect={vi.fn()} />);
    await user.click(screen.getByLabelText("Open emoji picker"));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("closes picker on second button click", async () => {
    const user = userEvent.setup();
    render(<EmojiPicker onSelect={vi.fn()} />);
    const button = screen.getByLabelText("Open emoji picker");
    await user.click(button);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    await user.click(button);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("calls onSelect when an emoji is clicked", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(<EmojiPicker onSelect={onSelect} />);
    await user.click(screen.getByLabelText("Open emoji picker"));
    // Use an emoji that isn't the first in its category (avoids duplicate with tab icon)
    await user.click(screen.getByText("😃"));
    expect(onSelect).toHaveBeenCalledWith("😃");
  });

  it("closes picker after selecting an emoji", async () => {
    const user = userEvent.setup();
    render(<EmojiPicker onSelect={vi.fn()} />);
    await user.click(screen.getByLabelText("Open emoji picker"));
    await user.click(screen.getByText("😃"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("switches emoji categories", async () => {
    const user = userEvent.setup();
    render(<EmojiPicker onSelect={vi.fn()} />);
    await user.click(screen.getByLabelText("Open emoji picker"));
    // Default category is Smileys, switch to Gestures
    await user.click(screen.getByLabelText("Gestures"));
    // Use an emoji that isn't the first in Gestures (avoids duplicate with tab icon)
    expect(screen.getByText("🤚")).toBeInTheDocument();
  });

  it("disables button when disabled prop is true", () => {
    render(<EmojiPicker onSelect={vi.fn()} disabled />);
    expect(screen.getByLabelText("Open emoji picker")).toBeDisabled();
  });

  it("closes on Escape key", async () => {
    const user = userEvent.setup();
    render(<EmojiPicker onSelect={vi.fn()} />);
    await user.click(screen.getByLabelText("Open emoji picker"));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes when clicking outside", async () => {
    const user = userEvent.setup();
    render(
      <div>
        <div data-testid="outside">outside</div>
        <EmojiPicker onSelect={vi.fn()} />
      </div>,
    );
    await user.click(screen.getByLabelText("Open emoji picker"));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    await user.click(screen.getByTestId("outside"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});

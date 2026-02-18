import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { UserContextMenu } from "./user-context-menu";

describe("UserContextMenu", () => {
  beforeEach(() => {
    if (!navigator.clipboard) {
      Object.defineProperty(navigator, "clipboard", {
        value: { writeText: vi.fn().mockResolvedValue(undefined) },
        writable: true,
        configurable: true,
      });
    } else {
      vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
    }
  });

  it("renders children", () => {
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );
    expect(screen.getByText("Alice")).toBeInTheDocument();
  });

  it("opens menu on click", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByText("Copy username")).toBeInTheDocument();
  });

  it("opens menu on right-click", () => {
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    fireEvent.contextMenu(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menu")).toBeInTheDocument();
  });

  it("opens menu on Enter key", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    const trigger = screen.getByRole("button", { name: "Actions for Alice" });
    trigger.focus();
    await user.keyboard("{Enter}");
    expect(screen.getByRole("menu")).toBeInTheDocument();
  });

  it("shows username header in menu", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    // Username appears as header in menu (in addition to the trigger)
    const aliceElements = screen.getAllByText("Alice");
    expect(aliceElements.length).toBeGreaterThanOrEqual(2);
  });

  it("shows Mention option when onMention is provided", async () => {
    const user = userEvent.setup();
    const onMention = vi.fn();
    render(
      <UserContextMenu username="Alice" onMention={onMention}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menuitem", { name: "Mention" })).toBeInTheDocument();
  });

  it("hides Mention option when onMention is not provided", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.queryByRole("menuitem", { name: "Mention" })).not.toBeInTheDocument();
  });

  it("calls onMention and closes menu when Mention is clicked", async () => {
    const user = userEvent.setup();
    const onMention = vi.fn();
    render(
      <UserContextMenu username="Alice" onMention={onMention}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    await user.click(screen.getByRole("menuitem", { name: "Mention" }));
    expect(onMention).toHaveBeenCalledWith("Alice");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("copies username to clipboard when Copy username is clicked", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    await user.click(screen.getByRole("menuitem", { name: "Copy username" }));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("Alice");
  });

  it("copies @mention to clipboard when Copy @mention is clicked", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    await user.click(screen.getByRole("menuitem", { name: /Copy @mention/ }));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("@Alice");
  });

  it("closes menu on Escape key", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menu")).toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("closes menu on click outside", async () => {
    const user = userEvent.setup();
    render(
      <div>
        <span data-testid="outside">Outside</span>
        <UserContextMenu username="Alice">
          <span>Alice</span>
        </UserContextMenu>
      </div>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menu")).toBeInTheDocument();

    await user.click(screen.getByTestId("outside"));
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("shows Kick option when onKick is provided", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice" onKick={vi.fn()}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menuitem", { name: "Kick" })).toBeInTheDocument();
  });

  it("shows Ban option when onBan is provided", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice" onBan={vi.fn()}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menuitem", { name: "Ban" })).toBeInTheDocument();
  });

  it("shows Mute option when onMute is provided", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice" onMute={vi.fn()}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.getByRole("menuitem", { name: "Mute" })).toBeInTheDocument();
  });

  it("hides moderation actions when callbacks are not provided", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice">
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    expect(screen.queryByRole("menuitem", { name: "Kick" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Ban" })).not.toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Mute" })).not.toBeInTheDocument();
  });

  it("calls onKick and closes menu when Kick is clicked", async () => {
    const user = userEvent.setup();
    const onKick = vi.fn();
    render(
      <UserContextMenu username="Alice" onKick={onKick}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    await user.click(screen.getByRole("menuitem", { name: "Kick" }));
    expect(onKick).toHaveBeenCalledWith("Alice");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("calls onBan and closes menu when Ban is clicked", async () => {
    const user = userEvent.setup();
    const onBan = vi.fn();
    render(
      <UserContextMenu username="Alice" onBan={onBan}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    await user.click(screen.getByRole("menuitem", { name: "Ban" }));
    expect(onBan).toHaveBeenCalledWith("Alice");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("calls onMute and closes menu when Mute is clicked", async () => {
    const user = userEvent.setup();
    const onMute = vi.fn();
    render(
      <UserContextMenu username="Alice" onMute={onMute}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    await user.click(screen.getByRole("menuitem", { name: "Mute" }));
    expect(onMute).toHaveBeenCalledWith("Alice");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("shows separator before moderation actions", async () => {
    const user = userEvent.setup();
    render(
      <UserContextMenu username="Alice" onKick={vi.fn()} onBan={vi.fn()} onMute={vi.fn()}>
        <span>Alice</span>
      </UserContextMenu>,
    );

    await user.click(screen.getByRole("button", { name: "Actions for Alice" }));
    const separators = screen.getAllByRole("separator");
    // One separator after username header, one before moderation actions
    expect(separators.length).toBe(2);
  });
});

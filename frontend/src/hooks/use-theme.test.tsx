import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeProvider, useTheme } from "./use-theme";

function TestConsumer() {
  const { theme, toggleTheme } = useTheme();
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <button onClick={toggleTheme}>Toggle</button>
    </div>
  );
}

describe("useTheme", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    document.documentElement.classList.remove("light");
  });

  it("throws when used outside ThemeProvider", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<TestConsumer />)).toThrow(
      "useTheme must be used within a ThemeProvider"
    );
    consoleSpy.mockRestore();
  });

  it("defaults to dark when no stored preference and system prefers dark", () => {
    document.documentElement.classList.add("dark");
    const original = window.matchMedia;
    window.matchMedia = vi.fn().mockReturnValue({ matches: false }) as typeof window.matchMedia;

    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    );

    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    window.matchMedia = original;
  });

  it("reads stored theme from localStorage", () => {
    localStorage.setItem("chatsphere-theme", "light");

    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    );

    expect(screen.getByTestId("theme")).toHaveTextContent("light");
  });

  it("toggles from dark to light", async () => {
    localStorage.setItem("chatsphere-theme", "dark");
    const user = userEvent.setup();

    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    );

    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);

    await user.click(screen.getByText("Toggle"));

    expect(screen.getByTestId("theme")).toHaveTextContent("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
    expect(localStorage.getItem("chatsphere-theme")).toBe("light");
  });

  it("toggles from light to dark", async () => {
    localStorage.setItem("chatsphere-theme", "light");
    const user = userEvent.setup();

    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    );

    await user.click(screen.getByText("Toggle"));

    expect(screen.getByTestId("theme")).toHaveTextContent("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(localStorage.getItem("chatsphere-theme")).toBe("dark");
  });

  it("persists theme to localStorage", async () => {
    localStorage.setItem("chatsphere-theme", "dark");
    const user = userEvent.setup();

    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>
    );

    await user.click(screen.getByText("Toggle"));
    expect(localStorage.getItem("chatsphere-theme")).toBe("light");

    await user.click(screen.getByText("Toggle"));
    expect(localStorage.getItem("chatsphere-theme")).toBe("dark");
  });
});

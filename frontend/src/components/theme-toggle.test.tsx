import { describe, it, expect, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ThemeToggle } from "./theme-toggle";
import { ThemeProvider } from "@/hooks/use-theme";

function renderWithTheme(initialTheme: "dark" | "light" = "dark") {
  localStorage.setItem("chatsphere-theme", initialTheme);
  if (initialTheme === "dark") {
    document.documentElement.classList.add("dark");
  } else {
    document.documentElement.classList.remove("dark");
  }
  return render(
    <ThemeProvider>
      <ThemeToggle />
    </ThemeProvider>
  );
}

describe("ThemeToggle", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
  });

  it("renders a button", () => {
    renderWithTheme();
    expect(screen.getByRole("button")).toBeInTheDocument();
  });

  it('shows "Switch to light mode" label when in dark mode', () => {
    renderWithTheme("dark");
    expect(screen.getByLabelText("Switch to light mode")).toBeInTheDocument();
  });

  it('shows "Switch to dark mode" label when in light mode', () => {
    renderWithTheme("light");
    expect(screen.getByLabelText("Switch to dark mode")).toBeInTheDocument();
  });

  it("toggles theme when clicked", async () => {
    const user = userEvent.setup();
    renderWithTheme("dark");

    const button = screen.getByRole("button");
    await user.click(button);

    expect(screen.getByLabelText("Switch to dark mode")).toBeInTheDocument();
  });

  it("renders an SVG icon", () => {
    renderWithTheme();
    const button = screen.getByRole("button");
    expect(button.querySelector("svg")).toBeInTheDocument();
  });
});

import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import App from "./App";

describe("App", () => {
  it("renders the heading", () => {
    render(<App />);
    expect(screen.getByRole("heading", { name: /chatsphere/i })).toBeInTheDocument();
  });

  it("renders the description", () => {
    render(<App />);
    expect(screen.getByText(/real-time anonymous chat rooms/i)).toBeInTheDocument();
  });

  it("renders the get started button", () => {
    render(<App />);
    expect(screen.getByRole("button", { name: /get started/i })).toBeInTheDocument();
  });
});

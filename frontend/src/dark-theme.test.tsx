import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { resolve } from "path";

describe("Dark theme configuration", () => {
  it("sets color-scheme: dark on :root", () => {
    const css = readFileSync(resolve(__dirname, "index.css"), "utf-8");
    expect(css).toMatch(/color-scheme:\s*dark/);
  });

  it("does not define a dark variant (dark-only theme)", () => {
    const css = readFileSync(resolve(__dirname, "index.css"), "utf-8");
    expect(css).not.toContain("@custom-variant dark");
  });

  it("defines dark background color in :root", () => {
    const css = readFileSync(resolve(__dirname, "index.css"), "utf-8");
    // oklch(0.145 0 0) is near-black
    expect(css).toMatch(/--background:\s*oklch\(0\.145/);
  });

  it("defines light foreground color in :root", () => {
    const css = readFileSync(resolve(__dirname, "index.css"), "utf-8");
    // oklch(0.985 0 0) is near-white
    expect(css).toMatch(/--foreground:\s*oklch\(0\.985/);
  });

  it("has dark class on html element", () => {
    const html = readFileSync(resolve(__dirname, "../index.html"), "utf-8");
    expect(html).toMatch(/<html[^>]*class="[^"]*dark[^"]*"/);
  });

  it("applies bg-background and text-foreground to body", () => {
    const css = readFileSync(resolve(__dirname, "index.css"), "utf-8");
    expect(css).toContain("bg-background");
    expect(css).toContain("text-foreground");
  });
});

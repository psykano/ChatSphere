import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { resolve } from "path";

describe("Dark theme configuration", () => {
  const css = readFileSync(resolve(__dirname, "index.css"), "utf-8");
  const html = readFileSync(resolve(__dirname, "../index.html"), "utf-8");

  it("sets color-scheme: dark on :root", () => {
    expect(css).toMatch(/color-scheme:\s*dark/);
  });

  it("does not define a dark variant (dark-only theme)", () => {
    expect(css).not.toContain("@custom-variant dark");
  });

  it("defines dark background color in :root", () => {
    // oklch(0.145 0 0) is near-black
    expect(css).toMatch(/--background:\s*oklch\(0\.145/);
  });

  it("defines light foreground color in :root", () => {
    // oklch(0.985 0 0) is near-white
    expect(css).toMatch(/--foreground:\s*oklch\(0\.985/);
  });

  it("has dark class on html element", () => {
    expect(html).toMatch(/<html[^>]*class="[^"]*dark[^"]*"/);
  });

  it("has color-scheme meta tag set to dark", () => {
    expect(html).toMatch(
      /<meta\s+name="color-scheme"\s+content="dark"\s*\/?>/
    );
  });

  it("applies bg-background and text-foreground to body", () => {
    expect(css).toContain("bg-background");
    expect(css).toContain("text-foreground");
  });

  it("does not contain a light theme or :root override for light mode", () => {
    expect(css).not.toMatch(/\.light\s*\{/);
    expect(css).not.toMatch(/@media\s*\(\s*prefers-color-scheme:\s*light\s*\)/);
  });

  it("maps Tailwind theme colors to CSS variables", () => {
    expect(css).toContain("--color-background: var(--background)");
    expect(css).toContain("--color-foreground: var(--foreground)");
    expect(css).toContain("--color-primary: var(--primary)");
    expect(css).toContain("--color-secondary: var(--secondary)");
    expect(css).toContain("--color-muted: var(--muted)");
    expect(css).toContain("--color-accent: var(--accent)");
    expect(css).toContain("--color-destructive: var(--destructive)");
    expect(css).toContain("--color-border: var(--border)");
    expect(css).toContain("--color-input: var(--input)");
    expect(css).toContain("--color-ring: var(--ring)");
  });

  it("defines border radius CSS variables", () => {
    expect(css).toMatch(/--radius:\s*0\.625rem/);
    expect(css).toContain("--radius-sm:");
    expect(css).toContain("--radius-md:");
    expect(css).toContain("--radius-lg:");
  });
});

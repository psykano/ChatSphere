import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { resolve } from "path";

describe("Theme configuration", () => {
  const css = readFileSync(resolve(__dirname, "index.css"), "utf-8");
  const html = readFileSync(resolve(__dirname, "../index.html"), "utf-8");

  it("sets color-scheme: light on :root for light theme defaults", () => {
    expect(css).toMatch(/color-scheme:\s*light/);
  });

  it("sets color-scheme: dark in .dark selector", () => {
    expect(css).toMatch(/\.dark\s*\{[^}]*color-scheme:\s*dark/s);
  });

  it("defines light background color in :root", () => {
    expect(css).toMatch(/:root\s*\{[^}]*--background:\s*oklch\(1 /s);
  });

  it("defines dark foreground color in :root", () => {
    expect(css).toMatch(/:root\s*\{[^}]*--foreground:\s*oklch\(0\.145/s);
  });

  it("defines dark background color in .dark", () => {
    expect(css).toMatch(/\.dark\s*\{[^}]*--background:\s*oklch\(0\.145/s);
  });

  it("defines light foreground color in .dark", () => {
    expect(css).toMatch(/\.dark\s*\{[^}]*--foreground:\s*oklch\(0\.985/s);
  });

  it("has dark class on html element by default", () => {
    expect(html).toMatch(/<html[^>]*class="[^"]*dark[^"]*"/);
  });

  it("has color-scheme meta tag supporting both themes", () => {
    expect(html).toMatch(
      /<meta\s+name="color-scheme"\s+content="dark light"\s*\/?>/
    );
  });

  it("applies bg-background and text-foreground to body", () => {
    expect(css).toContain("bg-background");
    expect(css).toContain("text-foreground");
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

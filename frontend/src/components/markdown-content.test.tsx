import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MarkdownContent } from "./markdown-content";

describe("MarkdownContent", () => {
  it("renders plain text", () => {
    render(<MarkdownContent content="Hello world" />);
    expect(screen.getByText("Hello world")).toBeInTheDocument();
  });

  it("renders bold text", () => {
    render(<MarkdownContent content="this is **bold**" />);
    const bold = screen.getByText("bold");
    expect(bold.tagName).toBe("STRONG");
  });

  it("renders italic text", () => {
    render(<MarkdownContent content="this is *italic*" />);
    const italic = screen.getByText("italic");
    expect(italic.tagName).toBe("EM");
  });

  it("renders inline code", () => {
    render(<MarkdownContent content="use `console.log`" />);
    const code = screen.getByText("console.log");
    expect(code.tagName).toBe("CODE");
  });

  it("renders code blocks", () => {
    render(<MarkdownContent content={"```js\nconst x = 1;\n```"} />);
    const code = screen.getByText("const x = 1;");
    expect(code.tagName).toBe("CODE");
    expect(code.className).toContain("language-js");
  });

  it("renders links with target _blank", () => {
    render(<MarkdownContent content="[click](https://example.com)" />);
    const link = screen.getByText("click");
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "https://example.com");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("renders unordered lists", () => {
    render(<MarkdownContent content={"- item one\n- item two"} />);
    expect(screen.getByText("item one")).toBeInTheDocument();
    expect(screen.getByText("item two")).toBeInTheDocument();
  });

  it("renders ordered lists", () => {
    render(<MarkdownContent content={"1. first\n2. second"} />);
    expect(screen.getByText("first")).toBeInTheDocument();
    expect(screen.getByText("second")).toBeInTheDocument();
  });

  it("renders blockquotes", () => {
    render(<MarkdownContent content="> quoted text" />);
    const quote = screen.getByText("quoted text");
    expect(quote.closest("blockquote")).toBeInTheDocument();
  });

  it("applies custom className", () => {
    const { container } = render(
      <MarkdownContent content="test" className="custom-class" />,
    );
    expect(container.firstChild).toHaveClass("custom-class");
  });
});

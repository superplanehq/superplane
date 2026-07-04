import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ConfigurationValueDisplay } from "./ConfigurationValueDisplay";
import type { ConfigurationDisplayRow } from "./types";

function renderRow(row: ConfigurationDisplayRow) {
  return render(<ConfigurationValueDisplay row={row} />);
}

describe("ConfigurationValueDisplay", () => {
  it("renders validated http(s) URLs as external links", () => {
    renderRow({
      key: "endpoint",
      label: "Endpoint",
      kind: "url",
      displayText: "https://api.example.com/hook",
      href: "https://api.example.com/hook",
    });

    const link = screen.getByRole("link", { name: "https://api.example.com/hook" });
    expect(link).toHaveAttribute("href", "https://api.example.com/hook");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("does not render javascript: URLs as links for url fields", () => {
    renderRow({
      key: "endpoint",
      label: "Endpoint",
      kind: "text",
      displayText: "javascript:alert(1)",
    });

    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("javascript:alert(1)").tagName).toBe("SPAN");
  });

  it("does not render links when href is an unsafe scheme even if kind is url", () => {
    renderRow({
      key: "endpoint",
      label: "Endpoint",
      kind: "url",
      displayText: "javascript:alert(1)",
      href: "javascript:alert(1)",
    });

    expect(screen.queryByRole("link")).not.toBeInTheDocument();
    expect(screen.getByText("javascript:alert(1)").tagName).toBe("SPAN");
  });

  it("preserves newlines in monospace code fallback", () => {
    renderRow({
      key: "script",
      label: "Script",
      kind: "code",
      displayText: "line one\nline two",
    });

    const value = screen.getByText(/line one/);
    expect(value).toHaveClass("whitespace-pre-wrap");
    expect(value.textContent).toBe("line one\nline two");
  });
});

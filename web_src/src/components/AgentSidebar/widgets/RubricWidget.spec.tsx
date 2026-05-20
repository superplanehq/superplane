import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RubricWidget } from "./RubricWidget";

describe("RubricWidget", () => {
  it("renders markdown for criteria in the inline preview", () => {
    render(
      <RubricWidget
        title="Test Plan"
        criteria={[{ text: "Use **bold** and `code` correctly" }]}
      />,
    );

    expect(screen.getByText("bold").tagName).toBe("STRONG");
    expect(screen.getByText("code").tagName).toBe("CODE");
  });

  it("preserves agent link protocols in rubric criterion markdown", () => {
    render(
      <RubricWidget
        title="Test Plan"
        criteria={[
          {
            text: "Navigate to [run link](run:123e4567-e89b-12d3-a456-426614174000) and confirm it renders.",
          },
        ]}
      />,
    );

    const link = screen.getByRole("link", { name: "run link" });
    expect(link.getAttribute("href")).toBe("run:123e4567-e89b-12d3-a456-426614174000");
  });

  it("wraps GFM tables in a horizontal overflow container (inline preview)", () => {
    const { container } = render(
      <RubricWidget
        title="Test Plan"
        criteria={[
          {
            text: ["| Column A | Column B |", "| --- | --- |", "| 1 | 2 |"].join("\n"),
          },
        ]}
      />,
    );

    const table = container.querySelector("table");
    expect(table).not.toBeNull();
    const wrapper = table?.closest("div");
    expect(wrapper).not.toBeNull();
    expect(wrapper?.className).toContain("my-4");
    expect(wrapper?.className).toContain("overflow-x-auto");
    expect(wrapper?.className).toContain("border-slate-200");
  });

  it("renders fenced code blocks inside the rubric widget (inline preview)", () => {
    render(
      <RubricWidget
        title="Test Plan"
        criteria={[{ text: "Run this:\n\n```bash\nnpm test\n```" }]}
      />,
    );

    // No modal opened — this exercises the inline preview only.
    const codeElement = screen.getByText("npm test", { selector: "code" });
    expect(codeElement.closest("pre")).not.toBeNull();
  });

  it("renders markdown for criteria inside a categorized inline preview", () => {
    render(
      <RubricWidget
        title="Test Plan"
        criteria={[{ text: "**Architecture**: use a queue" }]}
        categories={[
          {
            heading: "Architecture",
            criteria: [{ text: "**Use** a queue" }],
          },
        ]}
      />,
    );

    expect(screen.getByText("Use").tagName).toBe("STRONG");
  });

  it("renders markdown (including code blocks) inside the Full Plan modal", () => {
    render(
      <RubricWidget
        title="Test Plan"
        criteria={[{ text: "Run this:\n\n```bash\nnpm test\n```" }]}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /view full plan/i }));

    // Dialog content lives in a portal-less <div fixed>; query by role/heading.
    const heading = screen.getByRole("heading", { name: "Test Plan" });
    const modal = heading.closest("div.fixed") as HTMLElement;
    expect(modal).not.toBeNull();

    // The fenced code block must render as <pre><code>, not as raw backticks.
    const codeElement = within(modal).getByText("npm test", { selector: "code" });
    expect(codeElement.closest("pre")).not.toBeNull();
  });

  it("wraps GFM tables in a horizontal overflow container (Full Plan modal)", () => {
    render(
      <RubricWidget
        title="Test Plan"
        criteria={[
          {
            text: ["| Wide Column A | Wide Column B |", "| --- | --- |", "| 1 | 2 |"].join("\n"),
          },
        ]}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /view full plan/i }));

    const heading = screen.getByRole("heading", { name: "Test Plan" });
    const modal = heading.closest("div.fixed") as HTMLElement;
    expect(modal).not.toBeNull();

    const table = modal.querySelector("table");
    expect(table).not.toBeNull();
    const wrapper = table?.closest("div");
    expect(wrapper).not.toBeNull();
    expect(wrapper?.className).toContain("my-4");
    expect(wrapper?.className).toContain("overflow-x-auto");
    expect(wrapper?.className).toContain("border-slate-200");
  });
});

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
});

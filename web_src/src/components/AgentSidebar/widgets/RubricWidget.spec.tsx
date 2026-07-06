import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RubricWidget } from "./RubricWidget";

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "light", resolvedTheme: "light", setPreference: () => undefined }),
}));

describe("RubricWidget", () => {
  it("renders markdown for criteria in the inline preview", () => {
    render(<RubricWidget title="Test Plan" criteria={[{ text: "Use **bold** and `code` correctly" }]} />);

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
    render(<RubricWidget title="Test Plan" criteria={[{ text: "Run this:\n\n```bash\nnpm test\n```" }]} />);

    // No modal opened — this exercises the inline preview only.
    expect(screen.getByTestId("monaco-stub")).toHaveTextContent("npm test");
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
    const { container } = render(
      <RubricWidget title="Test Plan" criteria={[{ text: "Run this:\n\n```bash\nnpm test\n```" }]} />,
    );

    fireEvent.click(screen.getByRole("button", { name: /view full plan/i }));

    const modal = screen.getByRole("dialog", { name: "Test Plan" });
    expect(container.querySelector('[role="dialog"]')).not.toBeInTheDocument();

    expect(within(modal).getByTestId("monaco-stub")).toHaveTextContent("npm test");
  });

  it("keeps numbering aligned after body-rendered categories", () => {
    render(
      <RubricWidget
        title="Test Plan"
        criteria={[{ text: "First" }, { text: "Second" }, { text: "Fallback item" }]}
        categories={[
          {
            heading: "Body",
            criteria: [{ text: "First" }, { text: "Second" }],
            body: "1. First\n2. Second",
          },
          {
            heading: "Fallback",
            criteria: [{ text: "Fallback item" }],
          },
        ]}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /view full plan/i }));

    const fallbackItem = screen.getByText("Fallback item");
    const fallbackRow = fallbackItem.closest("div.flex") as HTMLElement;
    expect(fallbackRow).not.toBeNull();
    expect(within(fallbackRow).getByText("3.")).toBeInTheDocument();
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

    const modal = screen.getByRole("dialog", { name: "Test Plan" });

    const table = modal.querySelector("table");
    expect(table).not.toBeNull();
    const wrapper = table?.closest("div");
    expect(wrapper).not.toBeNull();
    expect(wrapper?.className).toContain("my-4");
    expect(wrapper?.className).toContain("overflow-x-auto");
    expect(wrapper?.className).toContain("border-slate-200");
  });
});

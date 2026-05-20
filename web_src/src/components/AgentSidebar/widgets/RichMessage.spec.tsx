import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { RichMessage } from "./RichMessage";

// Stub Monaco — irrelevant to layout-only assertions and slow to load.
vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

describe("RichMessage", () => {
  it("constrains its root so wide widgets can't expand a narrow parent", () => {
    const { container } = render(<RichMessage content="hello world" />);
    const root = container.firstChild as HTMLElement;

    // `w-full` + `min-w-0` is the pair that lets a flex child shrink below the
    // intrinsic width of inner widgets (charts, tables, monaco). Without these
    // a wide widget pushes the sidebar text into an x-scroll instead of wrapping.
    expect(root.className).toContain("w-full");
    expect(root.className).toContain("min-w-0");
  });

  it("keeps the markdown segment scoped to its parent width", () => {
    const { container } = render(<RichMessage content="some **markdown** here" />);
    const markdownContainer = container.querySelector("[class*='min-w-0']");
    expect(markdownContainer).not.toBeNull();
  });

  it("renders rubric agent links as in-app chips when ids are available", () => {
    render(
      <MemoryRouter>
        <RichMessage
          canvasId="canvas_123"
          organizationId="org_123"
          content={[
            ":::rubric Test Plan",
            "- Open the [run link](run:123e4567-e89b-12d3-a456-426614174000)",
            ":::",
          ].join("\n")}
        />
      </MemoryRouter>,
    );

    // When ids are available, `run:` links should render as RunChip buttons (not external anchors).
    expect(screen.getByRole("button", { name: "run link" })).toBeInTheDocument();
  });
});

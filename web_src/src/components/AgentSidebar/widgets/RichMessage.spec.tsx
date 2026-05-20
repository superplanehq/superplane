import { render } from "@testing-library/react";
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
});

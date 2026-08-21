import { fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { calcCodeBlockHeight } from "./calcCodeBlockHeight";
import { CodeBlockWidget } from "./CodeBlockWidget";

// Stub Monaco so the test doesn't try to spin up a real editor and so we can
// count how many times the inner editor mounts.
const editorMountSpy = vi.fn();
vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => {
    editorMountSpy();
    return <pre data-testid="monaco-stub">{value}</pre>;
  },
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "light", resolvedTheme: "light", setPreference: () => undefined }),
}));

describe("calcCodeBlockHeight", () => {
  it("fits one line without a tall minimum", () => {
    expect(calcCodeBlockHeight("echo hello")).toBe(35);
  });

  it("grows with extra lines and caps at 250", () => {
    expect(calcCodeBlockHeight("echo one\necho two")).toBe(54);
    expect(calcCodeBlockHeight(Array.from({ length: 40 }, (_, index) => `line ${index}`).join("\n"))).toBe(250);
  });
});

describe("CodeBlockWidget", () => {
  it("sizes a one-line snippet to the content height", () => {
    const { getByTestId } = render(<CodeBlockWidget code="echo hello" language="bash" />);

    expect(getByTestId("code-block-editor")).toHaveStyle({ height: "35px" });
  });

  it("applies width constraints so it cannot stretch a narrow parent", () => {
    const { container } = render(<CodeBlockWidget code="echo hello" language="bash" />);
    const root = container.firstChild as HTMLElement;

    expect(root.className).toContain("w-full");
    expect(root.className).toContain("min-w-0");
    // overflow-hidden keeps Monaco's internal layout from leaking past the box.
    expect(root.className).toContain("overflow-hidden");
  });

  it("does not re-render when an unrelated parent state changes", () => {
    editorMountSpy.mockClear();

    function Parent() {
      const [, setTick] = useState(0);
      return (
        <div>
          <button data-testid="bump" onClick={() => setTick((current) => current + 1)} />
          <CodeBlockWidget code="echo hello" language="bash" />
        </div>
      );
    }

    const { getByTestId } = render(<Parent />);
    const initialMounts = editorMountSpy.mock.calls.length;

    // Simulate the same kind of parent re-render that canvas zoom triggers.
    fireEvent.click(getByTestId("bump"));
    fireEvent.click(getByTestId("bump"));
    fireEvent.click(getByTestId("bump"));

    expect(editorMountSpy.mock.calls.length).toBe(initialMounts);
  });

  it("opens the shared fullscreen dialog when expand is clicked", () => {
    const { getByRole, getByText } = render(<CodeBlockWidget code="echo hello" language="bash" />);

    fireEvent.click(getByRole("button", { name: "Expand code" }));

    expect(getByText("BASH")).toBeInTheDocument();
    expect(getByRole("button", { name: "Copy" })).toBeInTheDocument();
  });
});

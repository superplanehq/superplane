import { fireEvent, render } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
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

describe("CodeBlockWidget", () => {
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
});

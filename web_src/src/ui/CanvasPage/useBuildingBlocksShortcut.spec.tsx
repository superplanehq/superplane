import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useBuildingBlocksShortcut } from "./useBuildingBlocksShortcut";

function Harness({
  disabled = false,
  isSidebarOpen = false,
  onOpen,
}: {
  disabled?: boolean;
  isSidebarOpen?: boolean;
  onOpen: () => void;
}) {
  useBuildingBlocksShortcut({ disabled, isSidebarOpen, onOpen });
  return (
    <div>
      <input data-testid="text-input" />
      <textarea data-testid="textarea" />
      <div data-testid="editable" contentEditable="true" />
      <div data-testid="monaco" className="monaco-editor">
        <div data-testid="monaco-inner" />
      </div>
    </div>
  );
}

describe("useBuildingBlocksShortcut", () => {
  it("calls onOpen when `c` is pressed with default body focus", () => {
    const onOpen = vi.fn();
    render(<Harness onOpen={onOpen} />);

    fireEvent.keyDown(window, { key: "c" });

    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("does not fire while disabled", () => {
    const onOpen = vi.fn();
    render(<Harness disabled={true} onOpen={onOpen} />);

    fireEvent.keyDown(window, { key: "c" });

    expect(onOpen).not.toHaveBeenCalled();
  });

  it("does not fire when the sidebar is already open", () => {
    const onOpen = vi.fn();
    render(<Harness isSidebarOpen={true} onOpen={onOpen} />);

    fireEvent.keyDown(window, { key: "c" });

    expect(onOpen).not.toHaveBeenCalled();
  });

  it("ignores modifier combinations like Cmd+C", () => {
    const onOpen = vi.fn();
    render(<Harness onOpen={onOpen} />);

    fireEvent.keyDown(window, { key: "c", metaKey: true });
    fireEvent.keyDown(window, { key: "c", ctrlKey: true });
    fireEvent.keyDown(window, { key: "c", altKey: true });

    expect(onOpen).not.toHaveBeenCalled();
  });

  it("ignores keys other than `c`", () => {
    const onOpen = vi.fn();
    render(<Harness onOpen={onOpen} />);

    fireEvent.keyDown(window, { key: "C" });
    fireEvent.keyDown(window, { key: "x" });

    expect(onOpen).not.toHaveBeenCalled();
  });

  it.each([
    ["text-input", "an <input>"],
    ["textarea", "a <textarea>"],
    ["editable", "a contenteditable element"],
    ["monaco-inner", "a Monaco editor"],
  ])("does not fire when focus is in %s (%s)", (testId) => {
    const onOpen = vi.fn();
    const { getByTestId } = render(<Harness onOpen={onOpen} />);

    fireEvent.keyDown(getByTestId(testId), { key: "c" });

    expect(onOpen).not.toHaveBeenCalled();
  });

  it("removes the listener on unmount so it cannot leak across canvases", () => {
    const onOpen = vi.fn();
    const { unmount } = render(<Harness onOpen={onOpen} />);

    unmount();
    fireEvent.keyDown(window, { key: "c" });

    expect(onOpen).not.toHaveBeenCalled();
  });
});

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useDeleteShortcut } from "./useDeleteShortcut";

function Harness({ disabled = false, onDelete }: { disabled?: boolean; onDelete: () => void }) {
  useDeleteShortcut({ disabled, onDelete });
  return (
    <div>
      <input data-testid="text-input" />
      <textarea data-testid="textarea" />
      <select data-testid="select">
        <option value="a">A</option>
      </select>
      <div data-testid="editable" contentEditable="true" />
      <div data-testid="monaco" className="monaco-editor">
        <div data-testid="monaco-inner" />
      </div>
    </div>
  );
}

describe("useDeleteShortcut", () => {
  it("calls onDelete when Delete is pressed with default body focus", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "Delete" });

    expect(onDelete).toHaveBeenCalledTimes(1);
  });

  it("calls onDelete when plain Backspace is pressed (Mac 'delete' key)", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "Backspace" });

    expect(onDelete).toHaveBeenCalledTimes(1);
  });

  it("does not fire when any modifier is held — those are OS-level text-editing shortcuts", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} />);

    for (const key of ["Delete", "Backspace"]) {
      fireEvent.keyDown(window, { key, metaKey: true });
      fireEvent.keyDown(window, { key, ctrlKey: true });
      fireEvent.keyDown(window, { key, altKey: true });
      fireEvent.keyDown(window, { key, shiftKey: true });
    }

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("does not fire while disabled", () => {
    const onDelete = vi.fn();
    render(<Harness disabled={true} onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "Delete" });
    fireEvent.keyDown(window, { key: "Backspace" });

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("ignores keys other than Delete or Backspace", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "x" });
    fireEvent.keyDown(window, { key: "Enter" });
    fireEvent.keyDown(window, { key: "Escape" });

    expect(onDelete).not.toHaveBeenCalled();
  });

  it.each([
    ["text-input", "an <input>"],
    ["textarea", "a <textarea>"],
    ["select", "a <select>"],
    ["editable", "a contenteditable element"],
    ["monaco-inner", "a Monaco editor"],
  ])("does not fire when focus is in %s (%s) — guards against #1668", (testId) => {
    const onDelete = vi.fn();
    const { getByTestId } = render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(getByTestId(testId), { key: "Delete" });
    fireEvent.keyDown(getByTestId(testId), { key: "Backspace" });

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("removes the listener on unmount so it cannot leak across canvases", () => {
    const onDelete = vi.fn();
    const { unmount } = render(<Harness onDelete={onDelete} />);

    unmount();
    fireEvent.keyDown(window, { key: "Delete" });

    expect(onDelete).not.toHaveBeenCalled();
  });
});

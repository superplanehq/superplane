import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useDeleteSelectionShortcut } from "./useDeleteSelectionShortcut";

function Harness({
  disabled = false,
  selectedNodeIds = ["node-1"],
  onDelete,
}: {
  disabled?: boolean;
  selectedNodeIds?: string[];
  onDelete: (nodeIds: string[]) => void;
}) {
  useDeleteSelectionShortcut({
    disabled,
    getSelectedNodeIds: () => selectedNodeIds,
    onDelete,
  });
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

describe("useDeleteSelectionShortcut", () => {
  it("deletes the selected nodes when Delete is pressed with default body focus", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} selectedNodeIds={["node-1", "node-2"]} />);

    fireEvent.keyDown(window, { key: "Delete" });

    expect(onDelete).toHaveBeenCalledWith(["node-1", "node-2"]);
  });

  it("deletes on Cmd+Backspace and Ctrl+Backspace", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "Backspace", metaKey: true });
    fireEvent.keyDown(window, { key: "Backspace", ctrlKey: true });

    expect(onDelete).toHaveBeenCalledTimes(2);
  });

  it("ignores a bare Backspace so it never deletes while editing text", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "Backspace" });

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("ignores Delete combined with a modifier", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "Delete", metaKey: true });
    fireEvent.keyDown(window, { key: "Delete", ctrlKey: true });
    fireEvent.keyDown(window, { key: "Delete", altKey: true });

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("does not fire while disabled", () => {
    const onDelete = vi.fn();
    render(<Harness disabled={true} onDelete={onDelete} />);

    fireEvent.keyDown(window, { key: "Delete" });

    expect(onDelete).not.toHaveBeenCalled();
  });

  it("does nothing when no nodes are selected", () => {
    const onDelete = vi.fn();
    render(<Harness onDelete={onDelete} selectedNodeIds={[]} />);

    fireEvent.keyDown(window, { key: "Delete" });

    expect(onDelete).not.toHaveBeenCalled();
  });

  it.each([
    ["text-input", "an <input>"],
    ["textarea", "a <textarea>"],
    ["editable", "a contenteditable element"],
    ["monaco-inner", "a Monaco editor"],
  ])("does not fire when focus is in %s (%s)", (testId) => {
    const onDelete = vi.fn();
    const { getByTestId } = render(<Harness onDelete={onDelete} />);

    fireEvent.keyDown(getByTestId(testId), { key: "Delete" });
    fireEvent.keyDown(getByTestId(testId), { key: "Backspace", metaKey: true });

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

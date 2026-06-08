import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { useCommandPaletteShortcut } from "./useCommandPaletteShortcut";

function Harness() {
  const { open, setOpen } = useCommandPaletteShortcut();
  return (
    <div>
      <span data-testid="open">{open ? "open" : "closed"}</span>
      <button onClick={() => setOpen(false)}>close</button>
      <input data-testid="text-input" />
      <textarea data-testid="textarea" />
      <div data-testid="editable" contentEditable="true" />
      <div data-testid="monaco" className="monaco-editor">
        <div data-testid="monaco-inner" />
      </div>
    </div>
  );
}

const stateOf = () => screen.getByTestId("open").textContent;

describe("useCommandPaletteShortcut", () => {
  it("opens on Cmd+K", () => {
    render(<Harness />);
    expect(stateOf()).toBe("closed");
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(stateOf()).toBe("open");
  });

  it("opens on Ctrl+K", () => {
    render(<Harness />);
    fireEvent.keyDown(window, { key: "k", ctrlKey: true });
    expect(stateOf()).toBe("open");
  });

  it("toggles closed when fired again", () => {
    render(<Harness />);
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(stateOf()).toBe("open");
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(stateOf()).toBe("closed");
  });

  it("ignores plain `k` without modifier", () => {
    render(<Harness />);
    fireEvent.keyDown(window, { key: "k" });
    expect(stateOf()).toBe("closed");
  });

  it("ignores Cmd+Shift+K and Cmd+Alt+K", () => {
    render(<Harness />);
    fireEvent.keyDown(window, { key: "k", metaKey: true, shiftKey: true });
    fireEvent.keyDown(window, { key: "k", metaKey: true, altKey: true });
    expect(stateOf()).toBe("closed");
  });

  it.each([
    ["text-input", "an <input>"],
    ["textarea", "a <textarea>"],
    ["editable", "a contenteditable element"],
    ["monaco-inner", "a Monaco editor"],
  ])("does not open when focus is in %s (%s)", (testId) => {
    render(<Harness />);
    const target = screen.getByTestId(testId);
    fireEvent.keyDown(target, { key: "k", metaKey: true });
    expect(stateOf()).toBe("closed");
  });

  it("still toggles closed from inside an editable element when already open", () => {
    render(<Harness />);
    fireEvent.keyDown(window, { key: "k", metaKey: true });
    expect(stateOf()).toBe("open");
    const input = screen.getByTestId("text-input");
    fireEvent.keyDown(input, { key: "k", metaKey: true });
    expect(stateOf()).toBe("closed");
  });
});

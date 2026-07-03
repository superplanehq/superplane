import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useCanvasToolSidebarState } from "./useCanvasToolSidebarState";

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => false, enabledExperimentalFeatures: [] }),
}));

function Harness({ onBeforeClose, canvasId }: { onBeforeClose: () => void; canvasId?: string }) {
  const state = useCanvasToolSidebarState({
    isEditing: false,
    readOnly: false,
    forceEnable: true,
    canvasId,
    onBeforeClose,
  });

  return (
    <div>
      <div data-testid="open-state">{state.isToolSidebarOpen ? "open" : "closed"}</div>
      <div data-testid="agent-state">{state.isAgentEnabled ? "enabled" : "disabled"}</div>
      <button type="button" onClick={state.handleToolSidebarToggle}>
        toggle
      </button>
    </div>
  );
}

describe("useCanvasToolSidebarState", () => {
  it("invokes onBeforeClose when toggling from open to closed", () => {
    window.localStorage.setItem("canvasAgentSidebarOpen", "true");
    const onBeforeClose = vi.fn();

    render(<Harness onBeforeClose={onBeforeClose} />);

    expect(screen.getByTestId("open-state")).toHaveTextContent("open");
    expect(screen.getByTestId("agent-state")).toHaveTextContent("disabled");

    fireEvent.click(screen.getByRole("button", { name: "toggle" }));

    expect(onBeforeClose).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("open-state")).toHaveTextContent("closed");
  });

  it("toggles the tool sidebar on Cmd/Ctrl+B outside editable fields", () => {
    window.localStorage.setItem("canvasAgentSidebarOpen", "false");
    const onBeforeClose = vi.fn();

    render(<Harness onBeforeClose={onBeforeClose} />);

    expect(screen.getByTestId("open-state")).toHaveTextContent("closed");

    fireEvent.keyDown(window, { key: "b", metaKey: true });

    expect(screen.getByTestId("open-state")).toHaveTextContent("open");
    expect(onBeforeClose).not.toHaveBeenCalled();

    fireEvent.keyDown(window, { key: "b", metaKey: true });

    expect(onBeforeClose).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("open-state")).toHaveTextContent("closed");
  });

  it("reads and persists the open state per canvas", () => {
    window.localStorage.setItem("canvasAgentSidebarOpen:canvas-a", "true");
    window.localStorage.setItem("canvasAgentSidebarOpen:canvas-b", "false");

    const { rerender } = render(<Harness onBeforeClose={vi.fn()} canvasId="canvas-a" />);
    expect(screen.getByTestId("open-state")).toHaveTextContent("open");

    rerender(<Harness onBeforeClose={vi.fn()} canvasId="canvas-b" />);
    expect(screen.getByTestId("open-state")).toHaveTextContent("closed");

    fireEvent.click(screen.getByRole("button", { name: "toggle" }));

    expect(screen.getByTestId("open-state")).toHaveTextContent("open");
    expect(window.localStorage.getItem("canvasAgentSidebarOpen:canvas-b")).toBe("true");
    // The other canvas preference is untouched.
    expect(window.localStorage.getItem("canvasAgentSidebarOpen:canvas-a")).toBe("true");
  });

  it("ignores Cmd/Ctrl+B while typing in an input", () => {
    window.localStorage.setItem("canvasAgentSidebarOpen", "false");
    const onBeforeClose = vi.fn();

    render(
      <div>
        <Harness onBeforeClose={onBeforeClose} />
        <input aria-label="search" />
      </div>,
    );

    const input = screen.getByRole("textbox", { name: "search" });
    input.focus();

    fireEvent.keyDown(input, { key: "b", metaKey: true });

    expect(screen.getByTestId("open-state")).toHaveTextContent("closed");
  });
});

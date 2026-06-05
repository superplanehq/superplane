import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useCanvasToolSidebarState } from "./useCanvasToolSidebarState";

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => false, enabledExperimentalFeatures: [] }),
}));

function Harness({ onBeforeClose }: { onBeforeClose: () => void }) {
  const state = useCanvasToolSidebarState({
    isEditing: false,
    readOnly: false,
    forceEnable: true,
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

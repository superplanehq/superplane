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

    fireEvent.click(screen.getByRole("button", { name: "toggle" }));

    expect(onBeforeClose).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("open-state")).toHaveTextContent("closed");
  });
});

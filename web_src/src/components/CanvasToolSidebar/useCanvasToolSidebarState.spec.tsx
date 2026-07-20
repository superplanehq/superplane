import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useCanvasToolSidebarState } from "./useCanvasToolSidebarState";

const featureFlags = vi.hoisted(() => ({ enabled: false }));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ has: () => featureFlags.enabled, enabledExperimentalFeatures: [] }),
}));

function Harness({
  onBeforeClose,
  canvasId,
  canUseAgents,
  forceEnable = true,
}: {
  onBeforeClose: () => void;
  canvasId?: string;
  canUseAgents?: boolean;
  forceEnable?: boolean;
}) {
  const state = useCanvasToolSidebarState({
    isEditing: false,
    readOnly: false,
    forceEnable,
    canvasId,
    canUseAgents,
    onBeforeClose,
  });

  return (
    <div>
      <div data-testid="open-state">{state.isToolSidebarOpen ? "open" : "closed"}</div>
      <div data-testid="agent-state">{state.isAgentEnabled ? "enabled" : "disabled"}</div>
      <div data-testid="agent-availability">{state.agentUnavailable ? "unavailable" : "available"}</div>
      <div data-testid="toggle-state">{state.showToolSidebarToggle ? "shown" : "hidden"}</div>
      <button type="button" onClick={state.handleToolSidebarToggle}>
        toggle
      </button>
      <button type="button" onClick={state.markAgentUnavailable}>
        mark-unavailable
      </button>
      <button type="button" onClick={state.markAgentAvailable}>
        mark-available
      </button>
    </div>
  );
}

describe("useCanvasToolSidebarState", () => {
  beforeEach(() => {
    featureFlags.enabled = false;
    window.localStorage.clear();
  });

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

  it("hides the agent surface when the user cannot use agents", () => {
    featureFlags.enabled = true;
    render(<Harness onBeforeClose={vi.fn()} canUseAgents={false} forceEnable={false} />);

    expect(screen.getByTestId("agent-state")).toHaveTextContent("disabled");
    expect(screen.getByTestId("toggle-state")).toHaveTextContent("hidden");
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

  it("keeps the sidebar open so the unavailable agent message can stay visible", () => {
    window.localStorage.setItem("canvasAgentSidebarOpen:canvas-x", "true");

    render(<Harness onBeforeClose={vi.fn()} canvasId="canvas-x" />);

    expect(screen.getByTestId("open-state")).toHaveTextContent("open");
    expect(screen.getByTestId("toggle-state")).toHaveTextContent("shown");

    fireEvent.click(screen.getByRole("button", { name: "mark-unavailable" }));

    expect(screen.getByTestId("toggle-state")).toHaveTextContent("shown");
    expect(screen.getByTestId("open-state")).toHaveTextContent("open");
    expect(window.localStorage.getItem("canvasAgentSidebarOpen:canvas-x")).toBe("true");
  });

  it("shows the toggle again and restores the stored preference when agent provisioning later succeeds", () => {
    window.localStorage.setItem("canvasAgentSidebarOpen:canvas-x", "true");
    render(<Harness onBeforeClose={vi.fn()} canvasId="canvas-x" />);

    fireEvent.click(screen.getByRole("button", { name: "mark-unavailable" }));
    expect(screen.getByTestId("toggle-state")).toHaveTextContent("shown");
    expect(screen.getByTestId("open-state")).toHaveTextContent("open");

    fireEvent.click(screen.getByRole("button", { name: "mark-available" }));

    expect(screen.getByTestId("toggle-state")).toHaveTextContent("shown");
    expect(screen.getByTestId("open-state")).toHaveTextContent("open");
  });

  it("rechecks agent availability when navigating back to a canvas", () => {
    const { rerender } = render(<Harness onBeforeClose={vi.fn()} canvasId="canvas-a" />);

    fireEvent.click(screen.getByRole("button", { name: "mark-unavailable" }));
    expect(screen.getByTestId("toggle-state")).toHaveTextContent("shown");
    expect(screen.getByTestId("agent-availability")).toHaveTextContent("unavailable");

    rerender(<Harness onBeforeClose={vi.fn()} canvasId="canvas-b" />);
    expect(screen.getByTestId("toggle-state")).toHaveTextContent("shown");
    expect(screen.getByTestId("agent-availability")).toHaveTextContent("available");

    rerender(<Harness onBeforeClose={vi.fn()} canvasId="canvas-a" />);
    expect(screen.getByTestId("toggle-state")).toHaveTextContent("shown");
    expect(screen.getByTestId("agent-availability")).toHaveTextContent("available");
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

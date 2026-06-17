import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { CanvasRunsSidebarTrigger } from "./CanvasRunsSidebarTrigger";

function makeRunsSidebarState(overrides: Partial<CanvasRunsSidebarState> = {}): CanvasRunsSidebarState {
  return {
    isRunsSidebarOpen: true,
    showRunsSidebarToggle: true,
    handleRunsSidebarToggle: vi.fn(),
    openRunsSidebar: vi.fn(),
    closeRunsSidebar: vi.fn(),
    ...overrides,
  };
}

describe("CanvasRunsSidebarTrigger", () => {
  it("does not render when the toggle is hidden", () => {
    render(<CanvasRunsSidebarTrigger runsSidebarState={makeRunsSidebarState({ showRunsSidebarToggle: false })} />);

    expect(screen.queryByTestId("canvas-runs-sidebar-toggle")).not.toBeInTheDocument();
  });

  it("toggles the runs sidebar", () => {
    const handleRunsSidebarToggle = vi.fn();

    render(<CanvasRunsSidebarTrigger runsSidebarState={makeRunsSidebarState({ handleRunsSidebarToggle })} />);

    fireEvent.click(screen.getByTestId("canvas-runs-sidebar-toggle"));

    expect(handleRunsSidebarToggle).toHaveBeenCalledTimes(1);
  });
});

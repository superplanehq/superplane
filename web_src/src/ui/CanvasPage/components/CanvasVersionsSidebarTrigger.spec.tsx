import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";
import { CanvasVersionsSidebarTrigger } from "./CanvasVersionsSidebarTrigger";

function makeVersionsSidebarState(overrides: Partial<CanvasVersionsSidebarState> = {}): CanvasVersionsSidebarState {
  return {
    isVersionsSidebarOpen: false,
    showVersionsSidebarToggle: true,
    handleVersionsSidebarToggle: vi.fn(),
    openVersionsSidebar: vi.fn(),
    closeVersionsSidebar: vi.fn(),
    ...overrides,
  };
}

describe("CanvasVersionsSidebarTrigger", () => {
  it("does not render when the toggle is hidden", () => {
    render(
      <CanvasVersionsSidebarTrigger
        versionsSidebarState={makeVersionsSidebarState({ showVersionsSidebarToggle: false })}
      />,
    );

    expect(screen.queryByTestId("canvas-versions-sidebar-toggle")).not.toBeInTheDocument();
  });

  it("toggles the versions sidebar", () => {
    const handleVersionsSidebarToggle = vi.fn();

    render(
      <CanvasVersionsSidebarTrigger versionsSidebarState={makeVersionsSidebarState({ handleVersionsSidebarToggle })} />,
    );

    fireEvent.click(screen.getByTestId("canvas-versions-sidebar-toggle"));

    expect(handleVersionsSidebarToggle).toHaveBeenCalledTimes(1);
  });
});

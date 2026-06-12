import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { useSidebarLayoutStore } from "@/stores/sidebarLayoutStore";
import { CanvasRunsSidebar } from ".";

function setViewportWidth(width: number): void {
  Object.defineProperty(window, "innerWidth", {
    configurable: true,
    writable: true,
    value: width,
  });
}

function renderRunsSidebar() {
  return render(
    <CanvasRunsSidebar isOpen>
      <div>Runs</div>
    </CanvasRunsSidebar>,
  );
}

describe("CanvasRunsSidebar", () => {
  beforeEach(() => {
    localStorage.clear();
    setViewportWidth(1280);
    useSidebarLayoutStore.getState().hydrateFromStorage();
  });

  it("starts with a compact default width", async () => {
    renderRunsSidebar();

    await waitFor(() => {
      expect(screen.getByTestId("canvas-runs-sidebar")).toHaveStyle({ width: "300px" });
    });
  });

  it("resizes from the drag start width instead of the moving left edge", async () => {
    setViewportWidth(1400);
    localStorage.setItem("runs-sidebar-width", "300");
    useSidebarLayoutStore.setState({
      leftMountCount: 1,
      leftWidth: 600,
      rightMountCount: 0,
      auxLeftMountCount: 0,
      auxLeftWidth: 300,
    });

    renderRunsSidebar();

    const sidebar = screen.getByTestId("canvas-runs-sidebar");
    sidebar.getBoundingClientRect = () =>
      ({
        left: useSidebarLayoutStore.getState().leftWidth,
        right: useSidebarLayoutStore.getState().leftWidth + useSidebarLayoutStore.getState().auxLeftWidth,
        top: 0,
        bottom: 0,
        width: useSidebarLayoutStore.getState().auxLeftWidth,
        height: 0,
        x: useSidebarLayoutStore.getState().leftWidth,
        y: 0,
        toJSON: () => ({}),
      }) as DOMRect;

    await waitFor(() => {
      expect(useSidebarLayoutStore.getState().auxLeftMountCount).toBe(1);
    });

    fireEvent.mouseDown(screen.getByTestId("canvas-runs-sidebar-resize-handle"), { clientX: 900 });
    await waitFor(() => {
      expect(useSidebarLayoutStore.getState().isAuxLeftResizing).toBe(true);
    });

    fireEvent.mouseMove(document, { clientX: 920 });
    useSidebarLayoutStore.setState({ leftWidth: 580 });
    fireEvent.mouseMove(document, { clientX: 920 });
    fireEvent.mouseUp(document);

    await waitFor(() => {
      const state = useSidebarLayoutStore.getState();
      expect(state.auxLeftWidth).toBe(320);
    });
  });
});

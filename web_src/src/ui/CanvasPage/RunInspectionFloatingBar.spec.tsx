import { fireEvent, render as testingLibraryRender, screen } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/contexts/ThemeProvider";

vi.mock("@/sentry", () => ({
  Sentry: {
    withScope: (callback: (scope: { setTag: typeof vi.fn; setExtra: typeof vi.fn }) => void) =>
      callback({
        setTag: vi.fn(),
        setExtra: vi.fn(),
      }),
    captureException: vi.fn(),
  },
}));

vi.mock("@xyflow/react", () => ({
  Background: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  Panel: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ReactFlow: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ReactFlowProvider: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ViewportPortal: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  useOnSelectionChange: vi.fn(),
  useReactFlow: vi.fn(() => ({
    fitView: vi.fn().mockResolvedValue(true),
    screenToFlowPosition: vi.fn((position) => position),
    getViewport: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
    setViewport: vi.fn(),
    getInternalNode: vi.fn(),
    zoomTo: vi.fn(),
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    getNodes: vi.fn(() => []),
    getZoom: vi.fn(() => 1),
  })),
  useStore: vi.fn((selector: (state: { minZoom: number; maxZoom: number }) => unknown) =>
    selector({ minZoom: 0.1, maxZoom: 1.5 }),
  ),
  useViewport: vi.fn(() => ({ zoom: 1, x: 0, y: 0 })),
}));

vi.mock("../BuildingBlocksSidebar", () => ({
  BuildingBlocksSidebar: () => null,
}));

vi.mock("../componentSidebar", () => ({
  ComponentSidebar: () => null,
}));

vi.mock("@/components/CanvasToolSidebar", () => ({
  CanvasToolSidebar: () => null,
}));

vi.mock("@/components/CanvasToolSidebar/useCanvasToolSidebarState", () => ({
  useCanvasToolSidebarState: () => ({
    canvasId: undefined,
    organizationId: undefined,
    isEditing: false,
    readOnly: false,
    isToolSidebarOpen: false,
    showToolSidebarToggle: false,
    handleToolSidebarToggle: vi.fn(),
    openToolSidebar: vi.fn(),
    closeToolSidebar: vi.fn(),
  }),
}));

vi.mock("./Header", () => ({
  Header: () => <header data-testid="canvas-header" />,
}));

import { CanvasPage } from "./index";

function render(ui: ReactElement) {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };

  return testingLibraryRender(ui, { wrapper: ThemeProvider });
}

describe("RunInspectionFloatingBar", () => {
  it("returns from run inspection to the live canvas", () => {
    const onBackToLiveCanvas = vi.fn();

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          onBackToLiveCanvas={onBackToLiveCanvas}
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
        />
      </MemoryRouter>,
    );

    expect(screen.getByText("Previewing previous run")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to Live Canvas" }));

    expect(onBackToLiveCanvas).toHaveBeenCalledTimes(1);
  });

  it("does not show the previous version bar while inspecting a run", () => {
    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          canvasStateMode="previewing-previous-version"
          isRunInspectionMode
          onBackToLiveCanvas={vi.fn()}
          onSeeCurrentVersion={vi.fn()}
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
        />
      </MemoryRouter>,
    );

    expect(screen.getByText("Previewing previous run")).toBeInTheDocument();
    expect(screen.queryByText("Previewing previous version")).not.toBeInTheDocument();
  });
});

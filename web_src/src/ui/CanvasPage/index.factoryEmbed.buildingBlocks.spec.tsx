import { act, render as testingLibraryRender, screen } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/contexts/ThemeProvider";

import { requestBuildingBlocksSidebar, subscribeBuildingBlocksSidebarChanged } from "./buildingBlocksSidebarRequest";
import { CANVAS_SIDEBAR_STORAGE_KEY } from "./index";

const { reactFlowPropsRef } = vi.hoisted(() => ({
  reactFlowPropsRef: {
    current: null as null | {
      nodes?: unknown;
    },
  },
}));

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
  Position: {
    Left: "left",
    Right: "right",
    Top: "top",
    Bottom: "bottom",
  },
  Background: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  Panel: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ReactFlow: (props: { children?: ReactNode; nodes?: unknown }) => {
    reactFlowPropsRef.current = props;
    return <div>{props.children}</div>;
  },
  ReactFlowProvider: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ViewportPortal: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  useOnSelectionChange: vi.fn(),
  useReactFlow: vi.fn(() => ({
    fitView: vi.fn(),
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
  BuildingBlocksSidebar: ({ isOpen, onToggle }: { isOpen: boolean; onToggle: (open: boolean) => void }) =>
    isOpen ? (
      <aside data-testid="building-blocks-sidebar">
        <button type="button" onClick={() => onToggle(false)}>
          Close
        </button>
      </aside>
    ) : null,
}));

vi.mock("../componentSidebar", () => ({
  ComponentSidebar: () => <aside data-testid="component-sidebar" />,
}));

vi.mock("@/components/CanvasToolSidebar", () => ({
  CanvasToolSidebar: () => null,
}));

vi.mock("@/pages/factories/agent/FactoryCanvasToolSidebar", () => ({
  FactoryCanvasToolSidebar: () => null,
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

const FACTORY_CANVAS_ID = "app-refund-implementer";

const existingNode = {
  id: "node-1",
  position: { x: 100, y: 100 },
  data: {
    label: "Filter",
    state: "pending" as const,
    type: "component",
  },
};

function render(ui: ReactElement) {
  return testingLibraryRender(ui, { wrapper: ThemeProvider });
}

describe("CanvasPage factory embed building blocks sidebar", () => {
  beforeEach(() => {
    reactFlowPropsRef.current = null;
    window.localStorage.removeItem(CANVAS_SIDEBAR_STORAGE_KEY);
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it("opens from a factory header request before the edit session is active", () => {
    const onChanged = vi.fn();
    const unsubscribe = subscribeBuildingBlocksSidebarChanged(onChanged);

    render(
      <MemoryRouter>
        <CanvasPage
          title="Refund Implementer"
          headerMode="version-live"
          canvasId={FACTORY_CANVAS_ID}
          nodes={[existingNode]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          factoryEmbed
          factoryEditWorkspace
          hideRightSideControls
          activeCanvasVersionId=""
        />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();

    act(() => {
      requestBuildingBlocksSidebar(FACTORY_CANVAS_ID, true);
    });

    expect(screen.getByTestId("building-blocks-sidebar")).toBeInTheDocument();
    expect(onChanged).not.toHaveBeenCalled();
    unsubscribe();
  });

  it("keeps the sidebar hidden in factory view-only when add controls are hidden", () => {
    render(
      <MemoryRouter>
        <CanvasPage
          title="Refund Implementer"
          headerMode="version-live"
          canvasId={FACTORY_CANVAS_ID}
          nodes={[existingNode]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          factoryEmbed
          hideAddControls
          hideRightSideControls
          activeCanvasVersionId=""
        />
      </MemoryRouter>,
    );

    act(() => {
      requestBuildingBlocksSidebar(FACTORY_CANVAS_ID, true);
    });

    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();
  });

  it("stays closed on an empty factory canvas until the header requests it", () => {
    render(
      <MemoryRouter>
        <CanvasPage
          title="Refund Implementer"
          headerMode="version-live"
          canvasId={FACTORY_CANVAS_ID}
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          factoryEmbed
          factoryEditWorkspace
          hideRightSideControls
          activeCanvasVersionId=""
        />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();
  });

  it("publishes a close when the user dismisses the panel", () => {
    const onChanged = vi.fn();
    const unsubscribe = subscribeBuildingBlocksSidebarChanged(onChanged);

    render(
      <MemoryRouter>
        <CanvasPage
          title="Refund Implementer"
          headerMode="version-live"
          canvasId={FACTORY_CANVAS_ID}
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          factoryEmbed
          factoryEditWorkspace
          hideRightSideControls
          activeCanvasVersionId=""
        />
      </MemoryRouter>,
    );

    act(() => {
      requestBuildingBlocksSidebar(FACTORY_CANVAS_ID, true);
    });

    expect(screen.getByTestId("building-blocks-sidebar")).toBeInTheDocument();

    act(() => {
      screen.getByRole("button", { name: "Close" }).click();
    });

    expect(onChanged).toHaveBeenCalledWith(FACTORY_CANVAS_ID, false);
    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();
    unsubscribe();
  });
});

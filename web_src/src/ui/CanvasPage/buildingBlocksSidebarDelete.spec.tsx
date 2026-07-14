import { act, render as testingLibraryRender, screen } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/contexts/ThemeProvider";

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
  BuildingBlocksSidebar: ({ isOpen }: { isOpen: boolean }) =>
    isOpen ? <aside data-testid="building-blocks-sidebar" /> : null,
}));

vi.mock("../componentSidebar", () => ({
  ComponentSidebar: () => <aside data-testid="component-sidebar" />,
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
  return testingLibraryRender(ui, { wrapper: ThemeProvider });
}

describe("CanvasPage building blocks sidebar delete", () => {
  beforeEach(() => {
    reactFlowPropsRef.current = null;
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it("closes the building blocks sidebar when a new placeholder component is deleted", () => {
    const onNodeDelete = vi.fn();
    const placeholderNode = {
      id: "placeholder-starter",
      position: { x: 100, y: 100 },
      data: {
        label: "New Component",
        state: "pending" as const,
        type: "component",
        outputChannels: ["default"],
        component: {
          title: "New Component",
          iconSlug: "plus",
          collapsed: false,
          includeEmptyState: true,
        },
      },
    };

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          nodes={[placeholderNode]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={true}
          activeCanvasVersionId="draft-version"
          onEdgeCreate={vi.fn()}
          onNodeDelete={onNodeDelete}
          workflowNodes={[{ id: "placeholder-starter", type: "TYPE_ACTION", name: "New Component" }]}
        />
      </MemoryRouter>,
    );

    const nodes = reactFlowPropsRef.current?.nodes as Array<{
      id: string;
      data: {
        _callbacksRef?: {
          current?: {
            handleNodeClick?: (nodeId: string) => void;
            onNodeDelete?: { current?: (nodeId: string) => void };
          };
        };
      };
    }>;

    act(() => {
      nodes[0].data._callbacksRef?.current?.handleNodeClick?.("placeholder-starter");
    });

    expect(screen.getByTestId("building-blocks-sidebar")).toBeInTheDocument();

    act(() => {
      nodes[0].data._callbacksRef?.current?.onNodeDelete?.current?.("placeholder-starter");
    });

    expect(screen.queryByTestId("building-blocks-sidebar")).not.toBeInTheDocument();
    expect(onNodeDelete).toHaveBeenCalledWith("placeholder-starter");
  });
});

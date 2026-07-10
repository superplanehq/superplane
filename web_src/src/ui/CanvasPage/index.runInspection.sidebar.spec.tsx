import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render as testingLibraryRender, screen, waitFor, fireEvent } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/contexts/ThemeProvider";

const { fitViewMock, getNodesMock, getViewportMock, reactFlowPropsRef, setViewportMock } = vi.hoisted(() => ({
  fitViewMock: vi.fn().mockResolvedValue(true),
  getNodesMock: vi.fn<() => Array<{ id: string; position: { x: number; y: number } }>>(() => []),
  getViewportMock: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
  reactFlowPropsRef: {
    current: null as null | {
      nodes?: unknown;
      onPaneClick?: (...args: unknown[]) => unknown;
      onInit?: (instance: { setViewport: (viewport: { x: number; y: number; zoom: number }) => void }) => unknown;
    },
  },
  setViewportMock: vi.fn(),
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
    fitView: fitViewMock,
    screenToFlowPosition: vi.fn((position) => position),
    getViewport: getViewportMock,
    setViewport: setViewportMock,
    getInternalNode: vi.fn(),
    zoomTo: vi.fn(),
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    getNodes: getNodesMock,
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

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: [] },
    isLoading: false,
  }),
}));

vi.mock("../componentSidebar", () => ({
  ComponentSidebar: () => <div data-testid="live-node-detail-pane-content" />,
}));

vi.mock("../Runs/RunInspectorPanel", () => ({
  RunInspectorPanel: ({ onClose, onEditNode }: { onClose?: () => void; onEditNode?: (nodeId: string) => void }) => (
    <div data-testid="run-inspector-panel">
      <button type="button" aria-label="Close" onClick={onClose} />
      <button type="button" onClick={() => onEditNode?.("run-node-1")}>
        Edit runtime config
      </button>
    </div>
  ),
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
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>{children}</ThemeProvider>
      </QueryClientProvider>
    );
  }

  return testingLibraryRender(ui, { wrapper: Wrapper });
}

describe("CanvasPage run inspection sidebar", () => {
  beforeEach(() => {
    reactFlowPropsRef.current = null;
    fitViewMock.mockClear();
    fitViewMock.mockResolvedValue(true);
    getNodesMock.mockReset();
    getNodesMock.mockReturnValue([]);
    getViewportMock.mockReset();
    getViewportMock.mockReturnValue({ x: 0, y: 0, zoom: 1 });
    setViewportMock.mockClear();
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it("shows the right run inspector during run inspection even when the live node inspector would be open", async () => {
    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          initialSidebar={{ isOpen: true, nodeId: "run-node-1" }}
          runNodeDetailNodeId="run-node-1"
          runNodeDetailCanvasId="canvas-1"
          runNodeDetailRun={{
            id: "run-1",
            rootEvent: { id: "root-event-1", nodeId: "trigger-node" },
          }}
          nodes={[
            {
              id: "run-node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Run node",
                state: "success",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("run-inspector-panel")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("live-node-detail-pane")).not.toBeInTheDocument();
  });

  it("closes only the right run inspector when the inspector close button is clicked", async () => {
    const onRunNodeDetailClose = vi.fn();
    const onBackToLiveCanvas = vi.fn();

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          runNodeDetailNodeId="run-node-1"
          runNodeDetailCanvasId="canvas-1"
          runNodeDetailRun={{
            id: "run-1",
            rootEvent: { id: "root-event-1", nodeId: "trigger-node" },
          }}
          onRunNodeDetailClose={onRunNodeDetailClose}
          onBackToLiveCanvas={onBackToLiveCanvas}
          nodes={[
            {
              id: "run-node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Run node",
                state: "success",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
        />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("run-inspector-panel")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Close" }));

    expect(onRunNodeDetailClose).toHaveBeenCalledOnce();
    expect(onBackToLiveCanvas).not.toHaveBeenCalled();
  });

  it("enters edit mode when editing runtime config from run inspection", async () => {
    const onEnterEditMode = vi.fn();
    const canvasProps = {
      title: "Canvas",
      headerMode: "version-live" as const,
      runNodeDetailNodeId: "run-node-1",
      runNodeDetailCanvasId: "canvas-1",
      runNodeDetailRun: {
        id: "run-1",
        rootEvent: { id: "root-event-1", nodeId: "trigger-node" },
      },
      nodes: [
        {
          id: "run-node-1",
          position: { x: 0, y: 0 },
          data: {
            label: "Run node",
            state: "success",
            type: "component",
          },
        },
      ],
      edges: [],
      buildingBlocks: [],
      activeCanvasVersionId: "live-version",
      onEnterEditMode,
    };

    render(
      <MemoryRouter>
        <CanvasPage {...canvasProps} isRunInspectionMode isEditing={false} />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByTestId("run-inspector-panel")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Edit runtime config" }));

    expect(onEnterEditMode).toHaveBeenCalledOnce();
    expect(screen.queryByTestId("live-node-detail-pane-content")).not.toBeInTheDocument();
  });

  it("shows right run inspector loading chrome while run detail is loading", () => {
    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          runCanvasLoading
          runNodeDetailCanvasId="canvas-1"
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole("complementary", { name: "Run inspector" })).toBeInTheDocument();
    expect(screen.queryByTestId("run-inspector-panel")).not.toBeInTheDocument();
  });

  it("does not open the right run inspector while editing", () => {
    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          runNodeDetailNodeId="run-node-1"
          runNodeDetailCanvasId="canvas-1"
          runNodeDetailRun={{
            id: "run-1",
            rootEvent: { id: "root-event-1", nodeId: "trigger-node" },
          }}
          nodes={[
            {
              id: "run-node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Run node",
                state: "success",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing
          activeCanvasVersionId="live-version"
        />
      </MemoryRouter>,
    );

    expect(screen.queryByTestId("run-inspector-panel")).not.toBeInTheDocument();
  });
});

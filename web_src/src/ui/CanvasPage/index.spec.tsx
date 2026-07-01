import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BlockData } from "./Block";

const { captureException, fitViewMock, getNodesMock, reactFlowPropsRef } = vi.hoisted(() => ({
  captureException: vi.fn(),
  fitViewMock: vi.fn().mockResolvedValue(true),
  getNodesMock: vi.fn<() => Array<{ id: string; position: { x: number; y: number } }>>(() => []),
  reactFlowPropsRef: {
    current: null as null | {
      nodes?: unknown;
      onConnectStart?: (...args: unknown[]) => unknown;
      onConnectEnd?: (...args: unknown[]) => unknown;
      onPaneClick?: (...args: unknown[]) => unknown;
      onEdgeMouseEnter?: (...args: unknown[]) => unknown;
      onEdgeMouseLeave?: (...args: unknown[]) => unknown;
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
    captureException,
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
    getViewport: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
    setViewport: vi.fn(),
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

import { CanvasNodeErrorBoundary, CanvasPage } from "./index";

function ThrowingNode(): ReactElement {
  throw new Error("render failed");
}

describe("CanvasNodeErrorBoundary", () => {
  beforeEach(() => {
    captureException.mockClear();
  });

  it("isolates one node render failure and keeps sibling nodes visible", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    render(
      <>
        <CanvasNodeErrorBoundary
          nodeId="node-1"
          nodeData={{ label: "Broken", state: "pending", type: "component" }}
          fallback={<div>node fallback</div>}
        >
          <ThrowingNode />
        </CanvasNodeErrorBoundary>
        <CanvasNodeErrorBoundary
          nodeId="node-2"
          nodeData={{ label: "Healthy", state: "pending", type: "component" }}
          fallback={<div>unused fallback</div>}
        >
          <div>healthy node</div>
        </CanvasNodeErrorBoundary>
      </>,
    );

    expect(screen.getByText("node fallback")).toBeInTheDocument();
    expect(screen.getByText("healthy node")).toBeInTheDocument();
    expect(captureException).toHaveBeenCalledTimes(1);
    consoleSpy.mockRestore();
  });

  it("does not retry a broken node when rerendered with equivalent data", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    const { rerender } = render(
      <CanvasNodeErrorBoundary
        nodeId="node-1"
        nodeData={{ label: "Broken", state: "pending", type: "component" }}
        fallback={<div>node fallback</div>}
      >
        <ThrowingNode />
      </CanvasNodeErrorBoundary>,
    );

    expect(screen.getByText("node fallback")).toBeInTheDocument();
    expect(captureException).toHaveBeenCalledTimes(1);

    rerender(
      <CanvasNodeErrorBoundary
        nodeId="node-1"
        nodeData={{ label: "Broken", state: "pending", type: "component" }}
        fallback={<div>node fallback</div>}
      >
        <ThrowingNode />
      </CanvasNodeErrorBoundary>,
    );

    expect(screen.getByText("node fallback")).toBeInTheDocument();
    expect(captureException).toHaveBeenCalledTimes(1);
    consoleSpy.mockRestore();
  });

  it("keeps the boundary alive when node data has an unknown runtime type", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const invalidNodeData = { label: "Broken", state: "pending", type: "unexpected" } as unknown as BlockData;

    render(
      <CanvasNodeErrorBoundary nodeId="node-unknown" nodeData={invalidNodeData} fallback={<div>unknown fallback</div>}>
        <ThrowingNode />
      </CanvasNodeErrorBoundary>,
    );

    expect(screen.getByText("unknown fallback")).toBeInTheDocument();
    expect(captureException).toHaveBeenCalledTimes(1);
    consoleSpy.mockRestore();
  });
});

describe("CanvasPage connection drop", () => {
  beforeEach(() => {
    reactFlowPropsRef.current = null;
    fitViewMock.mockClear();
    fitViewMock.mockResolvedValue(true);
    getNodesMock.mockReset();
    getNodesMock.mockReturnValue([]);
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it("does not close the building blocks sidebar from the pane click that follows a connection drop", () => {
    const onPlaceholderAdd = vi.fn(() => new Promise<string>(() => {}));

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={true}
          activeCanvasVersionId="draft-version"
          onEdgeCreate={vi.fn()}
          onPlaceholderAdd={onPlaceholderAdd}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("building-blocks-sidebar")).toBeInTheDocument();

    act(() => {
      reactFlowPropsRef.current?.onConnectStart?.(new MouseEvent("mousedown"), {
        nodeId: "source",
        handleId: "default",
        handleType: "source",
      });
      reactFlowPropsRef.current?.onConnectEnd?.(
        new MouseEvent("mouseup", {
          clientX: 100,
          clientY: 100,
        }),
      );
      reactFlowPropsRef.current?.onPaneClick?.();
    });

    expect(onPlaceholderAdd).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("building-blocks-sidebar")).toBeInTheDocument();
  });

  it("creates a placeholder when appending from an end-node connector", () => {
    const onPlaceholderAdd = vi.fn(async () => "placeholder-node");

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          nodes={[
            {
              id: "source-node",
              position: { x: 100, y: 200 },
              width: 240,
              data: {
                label: "Source",
                state: "pending",
                type: "component",
                outputChannels: ["default"],
                component: {
                  title: "Source",
                  iconSlug: "box",
                  collapsed: false,
                },
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={true}
          activeCanvasVersionId="draft-version"
          onEdgeCreate={vi.fn()}
          onPlaceholderAdd={onPlaceholderAdd}
        />
      </MemoryRouter>,
    );

    const nodes = reactFlowPropsRef.current?.nodes as Array<{
      data: {
        _callbacksRef?: {
          current?: {
            onAppendFromNode?: (nodeId: string, sourceHandleId?: string | null) => void;
          };
        };
      };
    }>;

    act(() => {
      nodes[0].data._callbacksRef?.current?.onAppendFromNode?.("source-node", "default");
    });

    expect(onPlaceholderAdd).toHaveBeenCalledWith({
      position: { x: 640, y: 200 },
      sourceNodeId: "source-node",
      sourceHandleId: "default",
    });
  });

  it("opens the building blocks sidebar without creating a placeholder when the add component button is clicked", async () => {
    const onPlaceholderAdd = vi.fn(
      async (_data: { position: { x: number; y: number }; sourceNodeId?: string; sourceHandleId?: string | null }) =>
        "placeholder-starter",
    );

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={true}
          activeCanvasVersionId="draft-version"
          onEdgeCreate={vi.fn()}
          onPlaceholderAdd={onPlaceholderAdd}
        />
      </MemoryRouter>,
    );

    await act(async () => {
      fireEvent.click(screen.getByTestId("canvas-add-component-button"));
    });

    expect(onPlaceholderAdd).not.toHaveBeenCalled();
    expect(screen.getByTestId("building-blocks-sidebar")).toBeInTheDocument();
  });

  it("loads node run data only while the component sidebar is open in live mode", async () => {
    const loadSidebarData = vi.fn();
    const getSidebarData = vi.fn(() => ({
      latestEvents: [],
      nextInQueueEvents: [],
      title: "Node",
      totalInQueueCount: 0,
      totalInHistoryCount: 0,
    }));

    const { rerender } = render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          canvasStateMode="editing"
          nodes={[
            {
              id: "node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={true}
          activeCanvasVersionId="draft-version"
          initialSidebar={{ isOpen: true, nodeId: "node-1" }}
          getSidebarData={getSidebarData}
          loadSidebarData={loadSidebarData}
          workflowNodes={[{ id: "node-1", type: "TYPE_ACTION", name: "Node" }]}
        />
      </MemoryRouter>,
    );

    await act(async () => {});
    expect(loadSidebarData).not.toHaveBeenCalled();

    rerender(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          canvasStateMode="default"
          nodes={[
            {
              id: "node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId=""
          initialSidebar={{ isOpen: true, nodeId: "node-1" }}
          getSidebarData={getSidebarData}
          loadSidebarData={loadSidebarData}
          workflowNodes={[{ id: "node-1", type: "TYPE_ACTION", name: "Node" }]}
        />
      </MemoryRouter>,
    );

    await waitFor(() => expect(loadSidebarData).toHaveBeenCalledWith("node-1"));
  });

  it("renders live inspector in bottom pane instead of right sidebar", async () => {
    const getSidebarData = vi.fn(() => ({
      latestEvents: [],
      nextInQueueEvents: [],
      title: "Node",
      totalInQueueCount: 0,
      totalInHistoryCount: 0,
    }));

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          canvasStateMode="default"
          nodes={[
            {
              id: "node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          initialSidebar={{ isOpen: true, nodeId: "node-1" }}
          getSidebarData={getSidebarData}
          workflowNodes={[{ id: "node-1", type: "TYPE_ACTION", name: "Node" }]}
        />
      </MemoryRouter>,
    );

    await act(async () => {});

    const bottomPane = screen.getByTestId("live-node-detail-pane");
    const componentSidebar = screen.getByTestId("component-sidebar");
    expect(bottomPane).toBeInTheDocument();
    expect(bottomPane).toContainElement(componentSidebar);
  });

  it("renders edit inspector in right sidebar, not bottom pane", async () => {
    const getSidebarData = vi.fn(() => ({
      latestEvents: [],
      nextInQueueEvents: [],
      title: "Node",
      totalInQueueCount: 0,
      totalInHistoryCount: 0,
    }));

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          canvasStateMode="editing"
          nodes={[
            {
              id: "node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={true}
          activeCanvasVersionId="draft-version"
          initialSidebar={{ isOpen: true, nodeId: "node-1" }}
          getSidebarData={getSidebarData}
          workflowNodes={[{ id: "node-1", type: "TYPE_ACTION", name: "Node" }]}
        />
      </MemoryRouter>,
    );

    await act(async () => {});

    expect(screen.queryByTestId("live-node-detail-pane")).not.toBeInTheDocument();
    expect(screen.getByTestId("component-sidebar")).toBeInTheDocument();
  });

  it("clears live bottom inspector selection from canvas pane click without closing the pane", async () => {
    const onSidebarChange = vi.fn();
    const getSidebarData = vi.fn(() => ({
      latestEvents: [],
      nextInQueueEvents: [],
      title: "Node",
      totalInQueueCount: 0,
      totalInHistoryCount: 0,
    }));

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          canvasStateMode="default"
          nodes={[
            {
              id: "node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          initialSidebar={{ isOpen: true, nodeId: "node-1" }}
          onSidebarChange={onSidebarChange}
          getSidebarData={getSidebarData}
          workflowNodes={[{ id: "node-1", type: "TYPE_ACTION", name: "Node" }]}
        />
      </MemoryRouter>,
    );

    await act(async () => {});

    expect(screen.getByTestId("component-sidebar")).toBeInTheDocument();

    act(() => {
      reactFlowPropsRef.current?.onPaneClick?.();
    });

    await waitFor(() => {
      expect(screen.getByTestId("live-bottom-inspector-empty")).toBeInTheDocument();
    });

    expect(onSidebarChange).toHaveBeenCalledWith(true, null);
    expect(screen.queryByTestId("component-sidebar")).not.toBeInTheDocument();
  });

  it("renders empty live bottom inspector when open without a selected node", async () => {
    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          canvasStateMode="default"
          nodes={[
            {
              id: "node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          initialSidebar={{ isOpen: true, nodeId: null }}
        />
      </MemoryRouter>,
    );

    await act(async () => {});

    expect(screen.getByTestId("live-node-detail-pane")).toBeInTheDocument();
    expect(screen.getByTestId("live-bottom-inspector-empty")).toBeInTheDocument();
    expect(screen.getByText("Select component to inspect")).toBeInTheDocument();
  });
});

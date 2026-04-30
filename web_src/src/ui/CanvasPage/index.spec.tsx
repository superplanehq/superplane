import { act, render, screen } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BlockData } from "./Block";

const { captureException, reactFlowPropsRef } = vi.hoisted(() => ({
  captureException: vi.fn(),
  reactFlowPropsRef: {
    current: null as null | {
      nodes?: unknown;
      onConnectStart?: (...args: unknown[]) => unknown;
      onConnectEnd?: (...args: unknown[]) => unknown;
      onPaneClick?: (...args: unknown[]) => unknown;
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
          nodes={[]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={true}
          activeCanvasVersionId="draft-version"
          onMemoryOpen={vi.fn()}
          onYamlOpen={vi.fn()}
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
          onMemoryOpen={vi.fn()}
          onYamlOpen={vi.fn()}
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
});

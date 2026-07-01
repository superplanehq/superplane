import { act, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { fitViewMock, getNodesMock, reactFlowPropsRef, setViewportMock } = vi.hoisted(() => ({
  fitViewMock: vi.fn().mockResolvedValue(true),
  getNodesMock: vi.fn<() => Array<{ id: string; position: { x: number; y: number } }>>(() => []),
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
    getViewport: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
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

describe("CanvasPage run inspection", () => {
  beforeEach(() => {
    reactFlowPropsRef.current = null;
    fitViewMock.mockClear();
    fitViewMock.mockResolvedValue(true);
    getNodesMock.mockReset();
    getNodesMock.mockReturnValue([]);
    setViewportMock.mockClear();
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it("does not re-run fit all when only run canvas nodes change", () => {
    vi.useFakeTimers();
    const hasFitToViewRef = { current: true };
    getNodesMock.mockReturnValue([
      {
        id: "run-node-1",
        position: { x: 0, y: 0 },
      },
    ]);

    const { rerender } = render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          nodes={[
            {
              id: "run-node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Run 1",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          hasFitToViewRef={hasFitToViewRef}
          fitAllRequest={0}
        />
      </MemoryRouter>,
    );

    act(() => {
      vi.runAllTimers();
    });

    expect(fitViewMock).toHaveBeenCalledTimes(1);

    getNodesMock.mockReturnValue([
      {
        id: "run-node-1",
        position: { x: 10, y: 20 },
      },
    ]);

    rerender(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          nodes={[
            {
              id: "run-node-1",
              position: { x: 10, y: 20 },
              data: {
                label: "Run 1",
                state: "success",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          hasFitToViewRef={hasFitToViewRef}
          fitAllRequest={0}
        />
      </MemoryRouter>,
    );

    act(() => {
      vi.runAllTimers();
    });

    expect(fitViewMock).toHaveBeenCalledTimes(1);
    vi.useRealTimers();
  });

  it("restores an existing run viewport on init without fitting all nodes", () => {
    const hasFitToViewRef = { current: true };
    const viewportRef = { current: { x: -240, y: -120, zoom: 0.75 } };

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          nodes={[
            {
              id: "run-node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Run 1",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          hasFitToViewRef={hasFitToViewRef}
          viewportRef={viewportRef}
        />
      </MemoryRouter>,
    );

    act(() => {
      reactFlowPropsRef.current?.onInit?.({ setViewport: setViewportMock });
    });

    expect(setViewportMock).toHaveBeenCalledWith({ x: -240, y: -120, zoom: 0.75 });
    expect(fitViewMock).not.toHaveBeenCalled();
  });

  it("refits when leaving run inspection with the same fit request nonce", () => {
    vi.useFakeTimers();
    const hasFitToViewRef = { current: true };
    getNodesMock.mockReturnValue([
      {
        id: "live-node-1",
        position: { x: 0, y: 0 },
      },
    ]);

    const { rerender } = render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          nodes={[
            {
              id: "live-node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Live node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          hasFitToViewRef={hasFitToViewRef}
          fitAllRequest={1}
        />
      </MemoryRouter>,
    );

    act(() => {
      vi.runAllTimers();
    });

    expect(fitViewMock).toHaveBeenCalledTimes(1);

    rerender(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode={false}
          nodes={[
            {
              id: "live-node-1",
              position: { x: 0, y: 0 },
              data: {
                label: "Live node",
                state: "pending",
                type: "component",
              },
            },
          ]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          hasFitToViewRef={hasFitToViewRef}
          fitAllRequest={1}
        />
      </MemoryRouter>,
    );

    act(() => {
      vi.runAllTimers();
    });

    expect(fitViewMock).toHaveBeenCalledTimes(2);
    vi.useRealTimers();
  });

  it("keeps the run node detail pane open and selected when the canvas background is clicked in runs mode", async () => {
    const onRunNodeDetailClose = vi.fn();
    const selectedRunNode = () =>
      (
        reactFlowPropsRef.current?.nodes as
          | Array<{
              id: string;
              selected?: boolean;
            }>
          | undefined
      )?.find((node) => node.id === "run-node-1");

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          runNodeDetailNodeId="run-node-1"
          onRunNodeDetailClose={onRunNodeDetailClose}
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
      expect(selectedRunNode()?.selected).toBe(true);
    });

    act(() => {
      reactFlowPropsRef.current?.onPaneClick?.();
    });

    expect(onRunNodeDetailClose).not.toHaveBeenCalled();
    expect(selectedRunNode()?.selected).toBe(true);
  });

  it("shows the run node detail pane during run inspection even when the live node inspector would be open", async () => {
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
      expect(screen.getByTestId("run-node-detail-pane")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("live-node-detail-pane")).not.toBeInTheDocument();
  });
});

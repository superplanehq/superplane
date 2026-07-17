import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render as testingLibraryRender, waitFor } from "@testing-library/react";
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

function selectedRunNodes() {
  return (reactFlowPropsRef.current?.nodes as Array<{ id: string; selected?: boolean }> | undefined) ?? [];
}

describe("CanvasPage auto-focus toggle", () => {
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

  it("skips fitView for focus requests when auto-focus is disabled but still selects the node", async () => {
    const hasFitToViewRef = { current: true };
    const initialViewport = { x: 10, y: 20, zoom: 0.8 };
    const viewportRef = { current: initialViewport };
    const runNode = {
      id: "run-node-1",
      position: { x: 0, y: 0 },
      data: { label: "Run 1", state: "pending", type: "component" },
    };
    const focusRequest = { nodeId: "run-node-1", requestId: 42, targetMode: "runs" as const, tab: "latest" as const };
    getNodesMock.mockReturnValue([runNode]);

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          isAutoFocusEnabled={false}
          nodes={[runNode]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          hasFitToViewRef={hasFitToViewRef}
          viewportRef={viewportRef}
          focusRequest={focusRequest}
        />
      </MemoryRouter>,
    );

    act(() => {
      reactFlowPropsRef.current?.onInit?.({ setViewport: setViewportMock });
    });

    await waitFor(() => {
      expect(selectedRunNodes().some((node) => node.id === "run-node-1" && node.selected)).toBe(true);
    });

    expect(fitViewMock).not.toHaveBeenCalled();
    expect(viewportRef.current).toEqual(initialViewport);
  });

  it("skips fitView for run participant fit requests when auto-focus is disabled and consumes the nonce", () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };
      const activeNode = { id: "run-node-1", position: { x: 0, y: 0 } };
      const renderedNodes = [{ ...activeNode, data: { label: "Run 1", state: "success", type: "component" } }];
      getNodesMock.mockReturnValue([activeNode]);

      const { rerender } = render(
        <MemoryRouter>
          <CanvasPage
            title="Canvas"
            headerMode="version-live"
            isRunInspectionMode
            isAutoFocusEnabled={false}
            nodes={renderedNodes}
            edges={[]}
            buildingBlocks={[]}
            isEditing={false}
            activeCanvasVersionId="live-version"
            hasFitToViewRef={hasFitToViewRef}
            viewportRef={viewportRef}
            fitAllRequest={1}
            fitAllFocusNodeIds={["run-node-1"]}
          />
        </MemoryRouter>,
      );

      act(() => {
        vi.runAllTimers();
      });

      expect(fitViewMock).not.toHaveBeenCalled();

      // Re-enabling auto-focus later must NOT retroactively fit for the already-consumed nonce.
      rerender(
        <MemoryRouter>
          <CanvasPage
            title="Canvas"
            headerMode="version-live"
            isRunInspectionMode
            isAutoFocusEnabled
            nodes={renderedNodes}
            edges={[]}
            buildingBlocks={[]}
            isEditing={false}
            activeCanvasVersionId="live-version"
            hasFitToViewRef={hasFitToViewRef}
            viewportRef={viewportRef}
            fitAllRequest={1}
            fitAllFocusNodeIds={["run-node-1"]}
          />
        </MemoryRouter>,
      );

      act(() => {
        vi.runAllTimers();
      });

      expect(fitViewMock).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("fits when a new run participant fit request arrives after auto-focus is re-enabled", () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };
      const activeNode = { id: "run-node-1", position: { x: 0, y: 0 } };
      const renderedNodes = [{ ...activeNode, data: { label: "Run 1", state: "success", type: "component" } }];
      getNodesMock.mockReturnValue([activeNode]);

      const { rerender } = render(
        <MemoryRouter>
          <CanvasPage
            title="Canvas"
            headerMode="version-live"
            isRunInspectionMode
            isAutoFocusEnabled={false}
            nodes={renderedNodes}
            edges={[]}
            buildingBlocks={[]}
            isEditing={false}
            activeCanvasVersionId="live-version"
            hasFitToViewRef={hasFitToViewRef}
            viewportRef={viewportRef}
            fitAllRequest={1}
            fitAllFocusNodeIds={["run-node-1"]}
          />
        </MemoryRouter>,
      );

      act(() => {
        vi.runAllTimers();
      });

      expect(fitViewMock).not.toHaveBeenCalled();

      rerender(
        <MemoryRouter>
          <CanvasPage
            title="Canvas"
            headerMode="version-live"
            isRunInspectionMode
            isAutoFocusEnabled
            nodes={renderedNodes}
            edges={[]}
            buildingBlocks={[]}
            isEditing={false}
            activeCanvasVersionId="live-version"
            hasFitToViewRef={hasFitToViewRef}
            viewportRef={viewportRef}
            fitAllRequest={2}
            fitAllFocusNodeIds={["run-node-1"]}
          />
        </MemoryRouter>,
      );

      act(() => {
        vi.runAllTimers();
      });

      expect(fitViewMock).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("fits for focus requests when auto-focus is enabled (default)", async () => {
    const hasFitToViewRef = { current: true };
    const initialViewport = { x: 10, y: 20, zoom: 0.8 };
    const focusedViewport = { x: -120, y: -80, zoom: 1.2 };
    const viewportRef = { current: initialViewport };
    const runNode = {
      id: "run-node-1",
      position: { x: 0, y: 0 },
      data: { label: "Run 1", state: "pending", type: "component" },
    };
    const focusRequest = { nodeId: "run-node-1", requestId: 3, targetMode: "runs" as const, tab: "latest" as const };
    getNodesMock.mockReturnValue([runNode]);
    getViewportMock.mockReturnValue(focusedViewport);

    render(
      <MemoryRouter>
        <CanvasPage
          title="Canvas"
          headerMode="version-live"
          isRunInspectionMode
          nodes={[runNode]}
          edges={[]}
          buildingBlocks={[]}
          isEditing={false}
          activeCanvasVersionId="live-version"
          hasFitToViewRef={hasFitToViewRef}
          viewportRef={viewportRef}
          focusRequest={focusRequest}
        />
      </MemoryRouter>,
    );

    act(() => {
      reactFlowPropsRef.current?.onInit?.({ setViewport: setViewportMock });
    });

    await waitFor(() => {
      expect(fitViewMock).toHaveBeenCalledTimes(1);
    });
    await waitFor(() => {
      expect(viewportRef.current).toEqual(focusedViewport);
    });
  });
});

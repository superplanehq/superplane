import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render as testingLibraryRender } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/contexts/ThemeProvider";

const { fitViewMock, getNodesMock, getViewportMock } = vi.hoisted(() => ({
  fitViewMock: vi.fn().mockResolvedValue(true),
  getNodesMock: vi.fn<() => Array<{ id: string; position: { x: number; y: number } }>>(() => []),
  getViewportMock: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
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
  ReactFlow: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ReactFlowProvider: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ViewportPortal: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  useOnSelectionChange: vi.fn(),
  useReactFlow: vi.fn(() => ({
    fitView: fitViewMock,
    screenToFlowPosition: vi.fn((position) => position),
    getViewport: getViewportMock,
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
  BuildingBlocksSidebar: () => null,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: [] },
    isLoading: false,
  }),
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
  Header: () => null,
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

describe("CanvasPage run participant fit", () => {
  beforeEach(() => {
    fitViewMock.mockClear();
    fitViewMock.mockResolvedValue(true);
    getNodesMock.mockReset();
    getNodesMock.mockReturnValue([]);
    getViewportMock.mockReset();
    getViewportMock.mockReturnValue({ x: 0, y: 0, zoom: 1 });
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it("fits run inspection to the active participant nodes when requested", async () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };
      const activeNode = { id: "run-node-1", position: { x: 0, y: 0 } };
      const inactiveNode = { id: "run-node-2", position: { x: 1000, y: 1000 } };
      const fittedViewport = { x: -120, y: -80, zoom: 1.1 };
      getNodesMock.mockReturnValue([activeNode, inactiveNode]);
      getViewportMock.mockReturnValue(fittedViewport);

      render(
        <MemoryRouter>
          <CanvasPage
            title="Canvas"
            headerMode="version-live"
            isRunInspectionMode
            nodes={[
              { ...activeNode, data: { label: "Run 1", state: "success", type: "component" } },
              { ...inactiveNode, data: { label: "Run 2", state: "pending", type: "component" } },
            ]}
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

      await act(async () => {
        await vi.runAllTimersAsync();
      });

      expect(fitViewMock).toHaveBeenCalledWith({
        nodes: [activeNode],
        includeHiddenNodes: true,
        maxZoom: 1.2,
        minZoom: 0.85,
        padding: 0.1,
        duration: 500,
      });
      expect(viewportRef.current).toEqual(fittedViewport);
    } finally {
      vi.useRealTimers();
    }
  });
});

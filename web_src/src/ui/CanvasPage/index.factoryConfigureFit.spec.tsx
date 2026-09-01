import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render as testingLibraryRender, screen } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { FACTORY_CONFIGURE_FIT_VIEW_OPTIONS } from "./canvasFitOptions";
import { FACTORY_CONFIGURE_FIT_SETTLE_MS } from "./factoryConfigureFitView";

const { fitViewMock, getViewportMock, getNodesMock, reactFlowPropsRef } = vi.hoisted(() => ({
  fitViewMock: vi.fn().mockResolvedValue(true),
  getViewportMock: vi.fn(() => ({ x: 0, y: 0, zoom: 1 })),
  getNodesMock: vi.fn(() => [] as Array<{ id: string }>),
  reactFlowPropsRef: {
    current: null as null | {
      onInit?: (instance: { setViewport: (viewport: unknown) => void }) => void;
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
  ReactFlow: (props: {
    children?: ReactNode;
    onInit?: (instance: { setViewport: (viewport: unknown) => void }) => void;
  }) => {
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
  Header: () => null,
}));

import { CanvasPage } from "./index";

const singleNode = [
  {
    id: "node-1",
    position: { x: 0, y: 0 },
    data: { label: "Node", state: "pending" as const, type: "component" as const },
  },
];

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

function canvasPage(overrides: {
  isEditing: boolean;
  factoryConfigure?: boolean;
  factoryConfigureLayoutReady?: boolean;
  initialSidebar?: { isOpen: boolean; nodeId: string };
  initialFocusNodeId?: string;
  hasFitToViewRef: { current: boolean };
  viewportRef: { current: { x: number; y: number; zoom: number } };
}) {
  return (
    <MemoryRouter>
      <CanvasPage
        title="Canvas"
        headerMode="version-live"
        nodes={singleNode}
        edges={[]}
        buildingBlocks={[]}
        activeCanvasVersionId="v1"
        factoryEditWorkspace
        {...overrides}
      />
    </MemoryRouter>
  );
}

describe("CanvasPage factory Configure fit", () => {
  beforeEach(() => {
    reactFlowPropsRef.current = null;
    fitViewMock.mockClear();
    fitViewMock.mockResolvedValue(true);
    getViewportMock.mockReset();
    getViewportMock.mockReturnValue({ x: 0, y: 0, zoom: 1 });
    getNodesMock.mockReset();
    getNodesMock.mockReturnValue([]);
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  it("fits the graph after Edit opens Configure", async () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 12, y: 8, zoom: 0.7 } };
      const fittedViewport = { x: -40, y: -20, zoom: 1 };
      getViewportMock.mockReturnValue(fittedViewport);

      const { rerender } = render(
        canvasPage({ isEditing: false, factoryConfigure: false, hasFitToViewRef, viewportRef }),
      );
      act(() => {
        reactFlowPropsRef.current?.onInit?.({ setViewport: vi.fn() });
      });
      fitViewMock.mockClear();

      rerender(canvasPage({ isEditing: true, factoryConfigure: true, hasFitToViewRef, viewportRef }));

      await act(async () => {
        await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_FIT_SETTLE_MS - 1);
      });
      expect(fitViewMock).not.toHaveBeenCalled();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(1);
      });

      expect(fitViewMock).toHaveBeenCalledTimes(1);
      expect(fitViewMock).toHaveBeenCalledWith({
        ...FACTORY_CONFIGURE_FIT_VIEW_OPTIONS,
        duration: 0,
      });
      expect(viewportRef.current).toEqual(fittedViewport);
      expect(screen.queryByTestId("factory-configure-enter-loading")).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it("covers the canvas until Configure fit finishes", async () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };

      const { rerender } = render(
        canvasPage({ isEditing: false, factoryConfigure: false, hasFitToViewRef, viewportRef }),
      );
      act(() => {
        reactFlowPropsRef.current?.onInit?.({ setViewport: vi.fn() });
      });

      rerender(canvasPage({ isEditing: true, factoryConfigure: true, hasFitToViewRef, viewportRef }));
      expect(screen.getByTestId("factory-configure-enter-loading")).toBeInTheDocument();
      expect(screen.getByText("Loading canvas...")).toBeInTheDocument();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_FIT_SETTLE_MS);
      });

      expect(screen.queryByTestId("factory-configure-enter-loading")).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it("keeps the cover until Configure layout snaps", async () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };

      render(
        canvasPage({
          isEditing: true,
          factoryConfigure: true,
          factoryConfigureLayoutReady: false,
          hasFitToViewRef,
          viewportRef,
        }),
      );
      act(() => {
        reactFlowPropsRef.current?.onInit?.({ setViewport: vi.fn() });
      });

      await act(async () => {
        await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_FIT_SETTLE_MS);
      });

      expect(fitViewMock).not.toHaveBeenCalled();
      expect(screen.getByTestId("factory-configure-enter-loading")).toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });

  it("does not fit while the automation stays in view mode", async () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };

      render(canvasPage({ isEditing: false, factoryConfigure: false, hasFitToViewRef, viewportRef }));
      act(() => {
        reactFlowPropsRef.current?.onInit?.({ setViewport: vi.fn() });
      });
      fitViewMock.mockClear();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_FIT_SETTLE_MS);
      });

      expect(fitViewMock).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("fits again on a later Configure visit", async () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };

      const { rerender } = render(
        canvasPage({ isEditing: true, factoryConfigure: true, hasFitToViewRef, viewportRef }),
      );
      act(() => {
        reactFlowPropsRef.current?.onInit?.({ setViewport: vi.fn() });
      });
      await act(async () => {
        await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_FIT_SETTLE_MS);
      });
      expect(fitViewMock).toHaveBeenCalledTimes(1);

      rerender(canvasPage({ isEditing: false, factoryConfigure: false, hasFitToViewRef, viewportRef }));
      fitViewMock.mockClear();

      rerender(canvasPage({ isEditing: true, factoryConfigure: true, hasFitToViewRef, viewportRef }));
      await act(async () => {
        await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_FIT_SETTLE_MS);
      });

      expect(fitViewMock).toHaveBeenCalledTimes(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("centers the deep-linked node when Configure opens from a selected component", async () => {
    vi.useFakeTimers();
    try {
      const hasFitToViewRef = { current: true };
      const viewportRef = { current: { x: 0, y: 0, zoom: 1 } };
      getNodesMock.mockReturnValue(singleNode);

      render(
        canvasPage({
          isEditing: true,
          factoryConfigure: true,
          initialSidebar: { isOpen: true, nodeId: "node-1" },
          hasFitToViewRef,
          viewportRef,
        }),
      );
      act(() => {
        reactFlowPropsRef.current?.onInit?.({ setViewport: vi.fn() });
      });
      fitViewMock.mockClear();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(FACTORY_CONFIGURE_FIT_SETTLE_MS);
      });

      expect(fitViewMock).toHaveBeenCalledTimes(1);
      expect(fitViewMock.mock.calls[0]?.[0]?.nodes?.[0]?.id).toBe("node-1");
      expect(fitViewMock.mock.calls[0]?.[0]?.minZoom).toBe(1);
      expect(fitViewMock.mock.calls[0]?.[0]?.maxZoom).toBe(1);
    } finally {
      vi.useRealTimers();
    }
  });
});

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render as testingLibraryRender } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";

const { reactFlowPropsRef } = vi.hoisted(() => ({
  reactFlowPropsRef: {
    current: null as null | {
      children?: ReactNode;
      isValidConnection?: (connection: { source: string | null; target: string | null }) => boolean;
      onConnect?: (connection: { source: string | null; target: string | null; sourceHandle: string | null }) => void;
    },
  },
}));

vi.mock("@/sentry", () => ({
  Sentry: {
    withScope: (callback: (scope: { setTag: typeof vi.fn; setExtra: typeof vi.fn }) => void) =>
      callback({ setTag: vi.fn(), setExtra: vi.fn() }),
    captureException: vi.fn(),
  },
}));

vi.mock("@xyflow/react", () => ({
  Background: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  Panel: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ReactFlow: (props: { children?: ReactNode }) => {
    reactFlowPropsRef.current = props;
    return <div>{props.children}</div>;
  },
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

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({ data: { executions: [] }, isLoading: false }),
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
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
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

function canvasNode(id: string, label: string, position: number) {
  return { id, position: { x: position, y: 0 }, data: { label, state: "success", type: "component" } };
}

const nodes = [canvasNode("deploy", "Deploy to Droplet", 0), canvasNode("deploy-failed", "Deploy Failed", 200)];

const edges = [{ id: "deploy->deploy-failed", source: "deploy", target: "deploy-failed" }];

const workflowNodes: ComponentsNode[] = [
  { id: "deploy", name: "Deploy to Droplet", type: "TYPE_ACTION", component: "noop" },
  { id: "deploy-failed", name: "Deploy Failed", type: "TYPE_ACTION", component: "noop" },
];

function renderCanvas(
  overrides: {
    onEdgeCreate?: (sourceId: string, targetId: string, sourceHandle?: string | null) => void;
  } = {},
) {
  render(
    <MemoryRouter>
      <CanvasPage
        title="Canvas"
        headerMode="default"
        activeCanvasVersionId="draft-version"
        isEditing
        nodes={nodes}
        edges={edges}
        workflowNodes={workflowNodes}
        buildingBlocks={[]}
        {...overrides}
      />
    </MemoryRouter>,
  );
}

describe("CanvasPage connection validation", () => {
  beforeEach(() => {
    reactFlowPropsRef.current = null;
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
  });

  /*
   * The repro from issue #5773: Deploy to Droplet -> Deploy Failed already
   * exists, so wiring Deploy Failed back into Deploy to Droplet is refused
   * while it is being dragged.
   */
  it("refuses a backward connection that would create a cycle", () => {
    renderCanvas({ onEdgeCreate: vi.fn() });

    const isValidConnection = reactFlowPropsRef.current?.isValidConnection;
    expect(isValidConnection).toBeDefined();
    expect(isValidConnection?.({ source: "deploy-failed", target: "deploy" })).toBe(false);
  });

  it("allows a forward connection", () => {
    renderCanvas({ onEdgeCreate: vi.fn() });

    expect(reactFlowPropsRef.current?.isValidConnection?.({ source: "deploy", target: "deploy-failed" })).toBe(true);
  });

  /*
   * isValidConnection blocks the drag interaction. onConnect re-checks so a
   * programmatic connection cannot slip an invalid edge past the guard.
   */
  it("does not create an edge when onConnect fires for a cycle-forming connection", () => {
    const onEdgeCreate = vi.fn();
    renderCanvas({ onEdgeCreate });

    reactFlowPropsRef.current?.onConnect?.({
      source: "deploy-failed",
      target: "deploy",
      sourceHandle: "default",
    });

    expect(onEdgeCreate).not.toHaveBeenCalled();
  });

  it("creates an edge when onConnect fires for a valid connection", () => {
    const onEdgeCreate = vi.fn();
    renderCanvas({ onEdgeCreate });

    reactFlowPropsRef.current?.onConnect?.({
      source: "deploy",
      target: "deploy-failed",
      sourceHandle: "default",
    });

    expect(onEdgeCreate).toHaveBeenCalledWith("deploy", "deploy-failed", "default");
  });
});

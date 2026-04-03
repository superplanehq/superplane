import { render, screen } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BlockData } from "./Block";

const { captureException } = vi.hoisted(() => ({
  captureException: vi.fn(),
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
  ReactFlow: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ReactFlowProvider: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  ViewportPortal: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  useOnSelectionChange: vi.fn(),
  useReactFlow: vi.fn(() => ({
    fitView: vi.fn(),
    screenToFlowPosition: vi.fn(),
    getZoom: vi.fn(() => 1),
  })),
  useViewport: vi.fn(() => ({ zoom: 1, x: 0, y: 0 })),
}));

import { CanvasNodeErrorBoundary } from "./index";

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

  it("isolates group node render failures", () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    render(
      <>
        <CanvasNodeErrorBoundary
          nodeId="group-1"
          nodeData={{ label: "Broken Group", state: "pending", type: "group" }}
          fallback={<div>group fallback</div>}
        >
          <ThrowingNode />
        </CanvasNodeErrorBoundary>
        <CanvasNodeErrorBoundary
          nodeId="node-3"
          nodeData={{ label: "Healthy", state: "pending", type: "component" }}
          fallback={<div>unused fallback</div>}
        >
          <div>another healthy node</div>
        </CanvasNodeErrorBoundary>
      </>,
    );

    expect(screen.getByText("group fallback")).toBeInTheDocument();
    expect(screen.getByText("another healthy node")).toBeInTheDocument();
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

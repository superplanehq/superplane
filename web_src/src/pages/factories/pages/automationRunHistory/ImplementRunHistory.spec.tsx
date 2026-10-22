import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { FactoryAutomationRunHistory } from "./FactoryAutomationRunHistory";

const hooks = vi.hoisted(() => ({
  useInfiniteCanvasRuns: vi.fn(),
  useDescribeRun: vi.fn(),
  useFactoryWorkOrders: vi.fn(),
}));
vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns: (...args: unknown[]) => hooks.useInfiniteCanvasRuns(...args),
  useDescribeRun: (...args: unknown[]) => hooks.useDescribeRun(...args),
  useEventExecutions: vi.fn(),
  useCanvasVersion: vi.fn(),
}));
vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: (...args: unknown[]) => hooks.useFactoryWorkOrders(...args),
}));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: () => {} }));

const history = (
  <FactoryAutomationRunHistory
    canvasId="canvas"
    organizationId="org"
    factoryId="factory"
    selectedRunId={null}
    onSelectRun={() => {}}
  />
);

describe("FactoryAutomationRunHistory", () => {
  it("shows the task key and title from the run list without loading archived tasks", () => {
    hooks.useInfiniteCanvasRuns.mockReturnValue({
      data: {
        pages: [
          {
            runs: [
              {
                id: "run-1",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                task: { id: "task-1", key: "SP-123", title: "Linked task" },
              },
            ],
          },
        ],
      },
      isLoading: false,
    });
    render(history);
    expect(hooks.useInfiniteCanvasRuns).toHaveBeenCalledWith("canvas", { includeTask: true });
    expect(hooks.useFactoryWorkOrders).not.toHaveBeenCalled();
    expect(screen.getByText("SP-123")).toBeInTheDocument();
    expect(screen.getByText("Linked task")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Loading task" })).not.toBeInTheDocument();
  });

  it("shows no linked task as soon as a run without a task loads", () => {
    hooks.useInfiniteCanvasRuns.mockReturnValue({
      data: { pages: [{ runs: [{ id: "run-1", state: "STATE_FINISHED", result: "RESULT_PASSED" }] }] },
      isLoading: false,
    });
    render(history);
    expect(screen.getByText("No linked task")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Loading task" })).not.toBeInTheDocument();
  });
});

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import type { FactoriesWorkOrder } from "@/api-client";
import { FactoryAutomationRunHistory } from "./FactoryAutomationRunHistory";

const queries = vi.hoisted(() => ({
  runs: {
    data: { pages: [{ runs: [{ id: "run-1", state: "STATE_FINISHED", result: "RESULT_PASSED" }] }] },
    isLoading: false,
  },
  tasks: { data: undefined as FactoriesWorkOrder[] | undefined, isLoading: true },
}));
vi.mock("@/hooks/useCanvasData", () => ({
  useInfiniteCanvasRuns: () => queries.runs,
  useDescribeRun: vi.fn(),
  useEventExecutions: vi.fn(),
  useCanvasVersion: vi.fn(),
}));
vi.mock("@/hooks/useFactoryData", () => ({ useFactoryWorkOrders: () => queries.tasks }));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: () => {} }));

describe("FactoryAutomationRunHistory", () => {
  it("shows fetched runs before task links finish loading, then enriches the rows", () => {
    const view = (
      <FactoryAutomationRunHistory
        canvasId="canvas"
        organizationId="org"
        factoryId="factory"
        selectedRunId={null}
        onSelectRun={() => {}}
      />
    );
    const { rerender } = render(view);
    expect(screen.getByRole("button", { name: "Open run run-1" })).toBeInTheDocument();
    expect(screen.queryByText("No task or source")).not.toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Loading task" })).toBeInTheDocument();
    queries.tasks.data = [{ id: "task-1", key: "SP-123", title: "Linked task", sourceRunId: "run-1" }];
    queries.tasks.isLoading = false;
    rerender(
      <FactoryAutomationRunHistory
        canvasId="canvas"
        organizationId="org"
        factoryId="factory"
        selectedRunId={null}
        onSelectRun={() => {}}
      />,
    );
    expect(screen.getByText("SP-123")).toBeInTheDocument();
    expect(screen.getByText("Linked task")).toBeInTheDocument();
  });
});

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";
import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { FactoryAutomationRunHistory } from "./FactoryAutomationRunHistory";

const api = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
  canvasesDescribeRun: vi.fn(),
  canvasesListEventExecutions: vi.fn(),
  canvasesDescribeCanvasVersion: vi.fn(),
}));
vi.mock("../../../../api-client/sdk.gen", () => api);
const taskData: { orders: FactoriesWorkOrder[] } = { orders: [] };
vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({ data: taskData.orders, isLoading: false }),
}));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: () => {} }));
vi.mock("@/ui/Runs/RunInspectorRunnerLogs", () => ({
  RunnerLogsTerminal: () => <p data-testid="runner-logs">Saved runner output</p>,
}));

describe("Implement run details hydration", () => {
  it("keeps loading feedback until cached run summaries and execution metadata are resolved", async () => {
    const run: CanvasesCanvasRun = {
      id: "run",
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      versionId: "version",
      createdAt: new Date().toISOString(),
      rootEvent: { id: "event" },
    };
    const execution: CanvasesCanvasNodeExecution = {
      id: "execution",
      runId: "run",
      nodeId: "implement",
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      metadata: { runner_broker_task_id: "task" },
    };
    const describe = deferred<{ data: { run: CanvasesCanvasRun } }>();
    const executions = deferred<{ data: { executions: CanvasesCanvasNodeExecution[] } }>();
    api.canvasesListRuns.mockResolvedValue({ data: { runs: [run], totalCount: 1 } });
    api.canvasesDescribeRun.mockReturnValue(describe.promise);
    api.canvasesListEventExecutions.mockReturnValue(executions.promise);
    api.canvasesDescribeCanvasVersion.mockResolvedValue({ data: { version: { spec: { nodes: [] } } } });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    client.setQueryData(canvasKeys.run("canvas", "run"), { run });
    client.setQueryData(canvasKeys.eventExecution("canvas", "event"), { executions: [] });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <FactoryAutomationRunHistory
            canvasId="canvas"
            organizationId="org"
            factoryId="factory"
            selectedRunId="run"
            onSelectRun={() => {}}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await waitFor(() => expect(api.canvasesDescribeRun).toHaveBeenCalled());
    expect(screen.queryByText("No logs are available for this activity.")).not.toBeInTheDocument();
    await act(async () => {
      describe.resolve({
        data: { run: { ...run, executions: [{ id: "execution", nodeId: "implement", result: "RESULT_FAILED" }] } },
      });
    });
    await waitFor(() => expect(api.canvasesListEventExecutions).toHaveBeenCalled());
    expect(screen.queryByText("No logs are available for this activity.")).not.toBeInTheDocument();
    await act(async () => {
      executions.resolve({ data: { executions: [execution] } });
    });
    await waitFor(() => expect(screen.getByTestId("runner-logs")).toBeInTheDocument());
    client.clear();
  });
  it("links a recorded existing task to its canonical task page", async () => {
    taskData.orders = [
      { id: "existing-task", key: "SP-42", number: "42", title: "Fix checkout", sourceRunId: "original-run" },
    ];
    const run: CanvasesCanvasRun = {
      id: "duplicate-run",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      versionId: "version",
      createdAt: new Date().toISOString(),
      rootEvent: {
        id: "duplicate-event",
        data: { type: "github.issue", data: { issue: { number: 42, title: "Fix checkout" } } },
      },
      executions: [
        {
          id: "create",
          nodeId: "create",
          componentName: "createWorkOrder",
          result: "RESULT_PASSED",
          existingTaskId: "existing-task",
        },
      ],
    };
    api.canvasesListRuns.mockResolvedValue({ data: { runs: [run], totalCount: 1 } });
    api.canvasesDescribeRun.mockResolvedValue({ data: { run } });
    api.canvasesListEventExecutions.mockResolvedValue({ data: { executions: [] } });
    api.canvasesDescribeCanvasVersion.mockResolvedValue({ data: { version: { spec: { nodes: [] } } } });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    try {
      render(
        <QueryClientProvider client={client}>
          <MemoryRouter initialEntries={["/org/workspaces/sp-abcd/lines/line-one"]}>
            <Routes>
              <Route
                path="/:organizationId/workspaces/:factoryKey/lines/:lineId"
                element={
                  <FactoryAutomationRunHistory
                    canvasId="canvas"
                    organizationId="org"
                    factoryId="factory"
                    selectedRunId="duplicate-run"
                    intakeSource="github-issues"
                    onSelectRun={() => {}}
                  />
                }
              />
              <Route path="/:organizationId/workspaces/:factoryKey/task/:orderNumber" element={<p>Task details</p>} />
            </Routes>
          </MemoryRouter>
        </QueryClientProvider>,
      );
      await waitFor(() =>
        expect(screen.getByRole("link", { name: "Task SP-42" })).toHaveAttribute(
          "href",
          "/org/workspaces/sp-abcd/task/42?lineId=line-one",
        ),
      );
      expect(screen.queryByText("Outcome unavailable")).not.toBeInTheDocument();
      fireEvent.click(screen.getByRole("link", { name: "Task SP-42" }));
      await waitFor(() => expect(screen.getByText("Task details")).toBeInTheDocument());
    } finally {
      taskData.orders = [];
      client.clear();
    }
  });
});

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

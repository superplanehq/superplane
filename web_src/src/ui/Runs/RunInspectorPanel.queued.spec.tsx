import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasNodeExecution, SuperplaneMeUser } from "@/api-client";
import { showSuccessToast } from "@/lib/toast";
import { executions, renderInspector, runningExecutions, runningRun } from "./RunInspectorPanel.spec.fixtures";

let mockedExecutions = executions;
let mockedExecutionsLoading = false;
let mockedMe: SuperplaneMeUser | null = null;

vi.mock("@uiw/react-json-view", () => ({
  default: ({ value, collapsed }: { value: unknown; collapsed?: boolean | number }) => (
    <pre data-testid="json-view" data-collapsed={String(collapsed)}>
      {JSON.stringify(value)}
    </pre>
  ),
}));

const reemitTriggerEventMock = vi.fn();
const cancelExecutionMock = vi.fn();
const cancelRunMock = vi.fn();
const invokeExecutionHookMock = vi.fn();
const describeRunMock = vi.fn();
const listNodeQueueItemsMock = vi.fn();
const deleteNodeQueueItemMock = vi.fn();

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitTriggerEventMock(...args),
    canvasesCancelExecution: (...args: unknown[]) => cancelExecutionMock(...args),
    canvasesCancelRun: (...args: unknown[]) => cancelRunMock(...args),
    canvasesInvokeNodeExecutionHook: (...args: unknown[]) => invokeExecutionHookMock(...args),
    canvasesDescribeRun: (...args: unknown[]) => describeRunMock(...args),
    canvasesListNodeQueueItems: (...args: unknown[]) => listNodeQueueItemsMock(...args),
    canvasesDeleteNodeQueueItem: (...args: unknown[]) => deleteNodeQueueItemMock(...args),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: mockedExecutions },
    isLoading: mockedExecutionsLoading,
  }),
  useCanvasVersion: () => ({
    data: undefined,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: mockedMe }),
}));

vi.mock("@/pages/app/mappers", () => ({
  getExecutionDetails: () => ({}),
  getState: () => (execution: CanvasesCanvasNodeExecution) =>
    execution.result === "RESULT_FAILED" ? "error" : "success",
  getStateMap: () => ({
    error: { badgeColor: "bg-red-500", label: "error" },
    success: { badgeColor: "bg-emerald-500", label: "success" },
    triggered: { badgeColor: "bg-blue-500", label: "triggered" },
  }),
  getTriggerRenderer: () => ({
    getTitleAndSubtitle: () => ({ title: "Deploy main", subtitle: "" }),
    getRootEventValues: () => ({ Source: "manual" }),
  }),
}));

vi.mock("@/pages/app/utils", () => ({
  buildEventInfo: (event: unknown) => event,
  buildExecutionInfo: (execution: unknown) => execution,
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

beforeEach(() => {
  mockedExecutions = executions;
  mockedExecutionsLoading = false;
  mockedMe = null;
  reemitTriggerEventMock.mockResolvedValue({});
  cancelExecutionMock.mockResolvedValue({});
  cancelRunMock.mockResolvedValue({});
  invokeExecutionHookMock.mockResolvedValue({});
  describeRunMock.mockResolvedValue({ data: { run: { queueItems: [] } } });
  listNodeQueueItemsMock.mockResolvedValue({ data: { items: [] } });
  deleteNodeQueueItemMock.mockResolvedValue({});
});

afterEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
});

describe("RunInspectorPanel queued steps", () => {
  it("jumps from a queued section to the failed execution for the same node", () => {
    mockedExecutions = executions.map((execution) =>
      execution.nodeId === "action-1"
        ? {
            ...execution,
            outputs: { default: [{ data: { error: "details" } }] },
          }
        : execution,
    );

    renderInspector({
      run: {
        ...runningRun,
        queueItems: [
          {
            id: "queue-failed-node",
            nodeId: "action-1",
            rootEvent: { id: "event-running", nodeId: "trigger-1" },
            input: { request: "retry after approval" },
            createdAt: "2026-05-01T12:00:05Z",
          },
        ],
      },
      selectedNodeId: "action-1",
    });

    const queuedHeader = screen.getByRole("heading", { name: /queued/i });
    fireEvent.click(within(queuedHeader).getByRole("button", { name: /Add Grade Label/i }));

    expect(screen.queryByRole("button", { name: /Input/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Output/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Jump to error" }));

    expect(screen.getByRole("button", { name: /Output/i })).toBeInTheDocument();
  });

  it("requests run cancellation from the stop action", async () => {
    mockedExecutions = runningExecutions;

    renderInspector({
      run: {
        ...runningRun,
        queueItems: [
          {
            id: "queue-1",
            nodeId: "action-2",
            rootEvent: { id: "event-running" },
            input: { approval: "pending" },
            createdAt: "2026-05-01T12:00:04Z",
          },
        ],
      },
    });

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);

    await waitFor(() => {
      expect(cancelRunMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            runId: "run-running",
          },
        }),
      );
    });

    expect(cancelRunMock).toHaveBeenCalledTimes(1);
    expect(cancelExecutionMock).not.toHaveBeenCalled();
    expect(deleteNodeQueueItemMock).not.toHaveBeenCalled();
  });

  it("requests run cancellation even when cached queue items are stale", async () => {
    mockedExecutions = executions;

    renderInspector({ run: { ...runningRun, queueItems: [] } });

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);

    await waitFor(() => {
      expect(cancelRunMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            runId: "run-running",
          },
        }),
      );
    });

    expect(describeRunMock).not.toHaveBeenCalled();
    expect(cancelRunMock).toHaveBeenCalledTimes(1);
  });

  it("requests run cancellation when cached queue items remain on the run", async () => {
    mockedExecutions = [];

    renderInspector({
      run: {
        ...runningRun,
        queueItems: [
          {
            id: "stale-queue-1",
            nodeId: "action-2",
            rootEvent: { id: "event-running" },
            input: { approval: "pending" },
          },
        ],
      },
    });

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);

    await waitFor(() => {
      expect(cancelRunMock).toHaveBeenCalled();
    });

    expect(showSuccessToast).toHaveBeenCalledWith("Run stopped");
  });

  it("requests run cancellation when cached running execution refs remain on the run", async () => {
    mockedExecutions = [];

    renderInspector({
      run: {
        ...runningRun,
        executions: [
          {
            id: "stale-execution-ref",
            nodeId: "action-2",
            state: "STATE_STARTED",
            result: "RESULT_UNKNOWN",
          },
        ],
      },
    });

    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);

    await waitFor(() => {
      expect(cancelRunMock).toHaveBeenCalled();
    });

    expect(showSuccessToast).toHaveBeenCalledWith("Run stopped");
  });

  it("renders queued items as non-expandable queued rows", () => {
    mockedExecutions = [];
    const onSelectNode = vi.fn();

    renderInspector({
      run: {
        ...runningRun,
        queueItems: [
          {
            id: "queue-approval",
            nodeId: "approval-1",
            rootEvent: { id: "event-running", nodeId: "trigger-1" },
            input: { request: "approve deploy" },
            createdAt: "2026-05-01T12:00:05Z",
          },
        ],
      },
      onSelectNode,
    });

    const queuedRow = screen.getByRole("button", { name: /Await Approval/i });
    expect(queuedRow).toBeInTheDocument();
    expect(queuedRow).not.toHaveAttribute("aria-expanded");
    expect(screen.getAllByText("queued").length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: /Input/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Runtime config/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Output/i })).not.toBeInTheDocument();

    fireEvent.click(queuedRow);

    expect(onSelectNode).toHaveBeenCalledWith("approval-1");
  });

  it("renders queued steps and allows stop while executions are loading", async () => {
    mockedExecutions = [];
    mockedExecutionsLoading = true;

    renderInspector({
      run: {
        ...runningRun,
        executions: [
          {
            id: "execution-ref-running",
            nodeId: "action-2",
            state: "STATE_STARTED",
            result: "RESULT_UNKNOWN",
          },
        ],
        queueItems: [
          {
            id: "queue-approval",
            nodeId: "approval-1",
            rootEvent: { id: "event-running", nodeId: "trigger-1" },
            input: { request: "approve deploy" },
            createdAt: "2026-05-01T12:00:05Z",
          },
        ],
      },
      selectedNodeId: "approval-1",
    });

    expect(screen.queryByText("Loading run steps...")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Await Approval/i })).toBeInTheDocument();

    const stopButton = screen.getAllByRole("button", { name: "Stop" })[0];
    expect(stopButton).toBeEnabled();
    fireEvent.click(stopButton);

    await waitFor(() => {
      expect(cancelRunMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            runId: "run-running",
          },
        }),
      );
    });

    expect(cancelRunMock).toHaveBeenCalledTimes(1);
    expect(cancelExecutionMock).not.toHaveBeenCalled();
    expect(deleteNodeQueueItemMock).not.toHaveBeenCalled();
  });

  it("cancels a queued step from the node accordion header", async () => {
    mockedExecutions = [];

    renderInspector({
      run: {
        ...runningRun,
        queueItems: [
          {
            id: "queue-approval",
            nodeId: "approval-1",
            rootEvent: { id: "event-running", nodeId: "trigger-1" },
            input: { request: "approve deploy" },
            createdAt: "2026-05-01T12:00:05Z",
          },
        ],
      },
      selectedNodeId: "approval-1",
    });

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() => {
      expect(deleteNodeQueueItemMock).toHaveBeenCalledWith(
        expect.objectContaining({
          path: {
            canvasId: "canvas-1",
            nodeId: "approval-1",
            itemId: "queue-approval",
          },
        }),
      );
    });
  });
});

import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { unmockedSrc } from "@/test/unmockedModule";
import { renderInspector, runningRun, workflowNodes } from "./RunInspectorPanel.spec.fixtures";

const reemitTriggerEvent = vi.fn();
const cancelRun = vi.fn();
const cancelExecution = vi.fn();
const invokeExecutionHook = vi.fn();
const deleteNodeQueueItem = vi.fn();

vi.mock("@uiw/react-json-view", () => ({
  default: () => <pre data-testid="json-view" />,
}));

vi.mock("@/api-client", () => {
  const actual = unmockedSrc<typeof ApiClient>("api-client");
  return {
    ...actual,
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitTriggerEvent(...args),
    canvasesCancelRun: (...args: unknown[]) => cancelRun(...args),
    canvasesCancelExecution: (...args: unknown[]) => cancelExecution(...args),
    canvasesInvokeNodeExecutionHook: (...args: unknown[]) => invokeExecutionHook(...args),
    canvasesDeleteNodeQueueItem: (...args: unknown[]) => deleteNodeQueueItem(...args),
    canvasesDescribeRun: vi.fn(),
    canvasesListNodeQueueItems: vi.fn(),
  };
});

vi.mock("@/hooks/useCanvasData", () => ({
  useEventExecutions: () => ({
    data: { executions: readOnlyExecutions },
    isLoading: false,
  }),
  useCanvasVersion: () => ({
    data: undefined,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: null }),
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

const readOnlyExecutions: CanvasesCanvasNodeExecution[] = [
  startedExecution("execution-running", "action-2"),
  startedExecution("execution-approval", "approval-1", {
    records: [{ index: 0, state: "pending", type: "user", user: { id: "account-1", email: "me@example.com" } }],
  }),
  startedExecution("execution-wait", "wait-1"),
];

function startedExecution(
  id: string,
  nodeId: string,
  metadata: CanvasesCanvasNodeExecution["metadata"] = {},
): CanvasesCanvasNodeExecution {
  return {
    id,
    nodeId,
    state: "STATE_STARTED",
    result: "RESULT_UNKNOWN",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    createdAt: "2026-05-01T12:00:03Z",
    updatedAt: "2026-05-01T12:00:04Z",
    outputs: {},
    metadata,
    configuration: {},
  };
}

describe("RunInspectorPanel read-only canvas", () => {
  beforeEach(() => {
    reemitTriggerEvent.mockReset();
    cancelRun.mockReset();
    cancelExecution.mockReset();
    invokeExecutionHook.mockReset();
    deleteNodeQueueItem.mockReset();
  });

  it("disables run actions when the user cannot update the canvas", async () => {
    const user = userEvent.setup();

    renderInspector({
      canUpdateCanvas: false,
      run: {
        ...runningRun,
        queueItems: [
          {
            id: "queue-1",
            nodeId: "action-1",
            rootEvent: { id: "event-running", nodeId: "trigger-1" },
            createdAt: "2026-05-01T12:00:05Z",
          },
        ],
      },
      workflowNodes: [...workflowNodes, { id: "wait-1", name: "Wait", type: "TYPE_ACTION", component: "wait" }],
      account: { id: "account-1", name: "Me", email: "me@example.com", avatar_url: "", installation_admin: false },
    });

    expectDisabled("Rerun");
    expectDisabled("Stop");
    expectDisabled("Cancel");
    expectDisabled("Approve");
    expectDisabled("Reject");
    expectDisabled("Push through");

    fireEvent.click(screen.getAllByRole("button", { name: "Rerun" })[0]);
    fireEvent.click(screen.getAllByRole("button", { name: "Stop" })[0]);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));
    fireEvent.click(screen.getByRole("button", { name: "Push through" }));

    expect(reemitTriggerEvent).not.toHaveBeenCalled();
    expect(cancelRun).not.toHaveBeenCalled();
    expect(cancelExecution).not.toHaveBeenCalled();
    expect(deleteNodeQueueItem).not.toHaveBeenCalled();
    expect(invokeExecutionHook).not.toHaveBeenCalled();

    await user.hover(screen.getAllByRole("button", { name: "Rerun" })[0]);
    expect(await screen.findByRole("tooltip")).toHaveTextContent("You need permission to update this canvas.");
  });
});

function expectDisabled(name: string) {
  const buttons = screen.getAllByRole("button", { name });
  expect(buttons.length).toBeGreaterThan(0);
  for (const button of buttons) {
    expect(button).toBeDisabled();
  }
}

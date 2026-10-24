import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";
import { unmockedSrc } from "@/test/unmockedModule";
import type { RunInspectorNodeSection } from "./types";
import { useRunInspectorActions } from "./useRunInspectorActions";

const reemitTriggerEvent = vi.hoisted(() => vi.fn());
const cancelRun = vi.hoisted(() => vi.fn());
const cancelExecution = vi.hoisted(() => vi.fn());
const deleteNodeQueueItem = vi.hoisted(() => vi.fn());
const invokeNodeExecutionHook = vi.hoisted(() => vi.fn());

vi.mock("@/api-client", () => {
  const actual = unmockedSrc<typeof ApiClient>("api-client");
  return {
    ...actual,
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitTriggerEvent(...args),
    canvasesCancelRun: (...args: unknown[]) => cancelRun(...args),
    canvasesCancelExecution: (...args: unknown[]) => cancelExecution(...args),
    canvasesDeleteNodeQueueItem: (...args: unknown[]) => deleteNodeQueueItem(...args),
    canvasesInvokeNodeExecutionHook: (...args: unknown[]) => invokeNodeExecutionHook(...args),
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

const run: CanvasesCanvasRun = {
  rootEvent: {
    id: "root-event-1",
    nodeId: "trigger-1",
  },
};

function renderActions(
  sections: RunInspectorNodeSection[],
  {
    executionsLoading = false,
    runOverride = run,
    canUpdateCanvas = true,
  }: { executionsLoading?: boolean; runOverride?: CanvasesCanvasRun; canUpdateCanvas?: boolean } = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return renderHook(
    () =>
      useRunInspectorActions({
        canvasId: "canvas-1",
        run: runOverride,
        sections,
        executionsLoading,
        canUpdateCanvas,
      }),
    {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      ),
    },
  );
}

function actionSection(overrides: Partial<RunInspectorNodeSection> = {}): RunInspectorNodeSection {
  return {
    sectionValue: "action-1",
    nodeId: "action-1",
    nodeName: "Action 1",
    isTrigger: false,
    isQueued: false,
    badge: null,
    tabData: null,
    upstreamSections: [],
    outputSections: [],
    actions: {
      canStop: false,
      canPushThrough: false,
      approvalRecords: [],
    },
    configurationFields: [],
    ...overrides,
  };
}

describe("useRunInspectorActions", () => {
  beforeEach(() => {
    reemitTriggerEvent.mockReset();
    cancelRun.mockReset();
    cancelExecution.mockReset();
    deleteNodeQueueItem.mockReset();
    invokeNodeExecutionHook.mockReset();
    reemitTriggerEvent.mockResolvedValue({ data: { eventId: "event-rerun" } });
    cancelRun.mockResolvedValue({});
    cancelExecution.mockResolvedValue({});
    deleteNodeQueueItem.mockResolvedValue({});
    invokeNodeExecutionHook.mockResolvedValue({});
  });

  it("does not call run actions without canvas update permission", () => {
    const section = actionSection({
      execution: {
        id: "execution-1",
        nodeId: "action-1",
        state: "STATE_STARTED",
      },
      queueItem: {
        id: "queue-1",
        nodeId: "action-1",
      },
    });
    const { result } = renderActions([section], {
      canUpdateCanvas: false,
      runOverride: { ...run, id: "run-1" },
    });

    expect(result.current.stopDisabled).toBe(true);
    expect(result.current.rerunDisabled).toBe(true);

    act(() => {
      result.current.rerun();
      result.current.stop();
      result.current.stopNode(section);
      result.current.cancelQueuedItem(section);
      result.current.invokeNodeHook(section, "approve");
    });

    expect(reemitTriggerEvent).not.toHaveBeenCalled();
    expect(cancelRun).not.toHaveBeenCalled();
    expect(cancelExecution).not.toHaveBeenCalled();
    expect(deleteNodeQueueItem).not.toHaveBeenCalled();
    expect(invokeNodeExecutionHook).not.toHaveBeenCalled();
  });

  it("allows Stop when action sections only have lightweight running execution refs", () => {
    const { result } = renderActions([
      actionSection({
        executionRef: {
          id: "execution-ref-1",
          nodeId: "action-1",
          state: "STATE_STARTED",
        },
      }),
    ]);

    expect(result.current.stopDisabled).toBe(false);
  });

  it("allows Stop for loaded action execution details so queued steps can be cancelled", () => {
    const { result } = renderActions([
      actionSection({
        execution: {
          id: "execution-1",
          nodeId: "action-1",
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
        },
      }),
    ]);

    expect(result.current.stopDisabled).toBe(false);
  });

  it("allows Stop for queued items while executions are loading", () => {
    const { result } = renderActions([], {
      executionsLoading: true,
      runOverride: {
        ...run,
        queueItems: [{ id: "queue-1", nodeId: "action-1" }],
      },
    });

    expect(result.current.stopDisabled).toBe(false);
  });
});

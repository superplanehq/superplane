import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type * as ApiClient from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";
import type { RunInspectorNodeSection } from "./types";
import { useRunInspectorActions } from "./useRunInspectorActions";

const reemitTriggerEventMock = vi.fn();

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof ApiClient>();
  return {
    ...actual,
    canvasesReemitTriggerEvent: (...args: unknown[]) => reemitTriggerEventMock(...args),
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

beforeEach(() => {
  reemitTriggerEventMock.mockReset();
  reemitTriggerEventMock.mockResolvedValue({ data: { eventId: "event-1" } });
});

afterEach(() => {
  vi.clearAllMocks();
});

function renderActions(
  sections: RunInspectorNodeSection[],
  {
    executionsLoading = false,
    runOverride = run,
    canRerun = true,
  }: { executionsLoading?: boolean; runOverride?: CanvasesCanvasRun; canRerun?: boolean } = {},
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
        canRerun,
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

  it("reflects the canRerun flag passed in", () => {
    const allowed = renderActions([], { canRerun: true });
    expect(allowed.result.current.canRerun).toBe(true);

    const denied = renderActions([], { canRerun: false });
    expect(denied.result.current.canRerun).toBe(false);
  });

  it("never calls the reemit API and logs a readable message when rerun is attempted without permission", async () => {
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { result } = renderActions([], { canRerun: false });

    act(() => {
      result.current.rerun();
    });

    await waitFor(() => expect(result.current.rerunPending).toBe(false));

    expect(reemitTriggerEventMock).not.toHaveBeenCalled();
    expect(consoleErrorSpy).toHaveBeenCalledWith(
      expect.stringContaining("You do not have permission to restart this run."),
    );

    consoleErrorSpy.mockRestore();
  });

  it("logs a readable message instead of [object Object] when the reemit request fails", async () => {
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    reemitTriggerEventMock.mockRejectedValueOnce({ code: 5, message: "Not found" });

    const { result } = renderActions([], { canRerun: true });

    act(() => {
      result.current.rerun();
    });

    await waitFor(() => expect(result.current.rerunPending).toBe(false));

    expect(reemitTriggerEventMock).toHaveBeenCalledTimes(1);
    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
    const [loggedMessage] = consoleErrorSpy.mock.calls[0];
    expect(loggedMessage).toContain("Not found");
    expect(loggedMessage).not.toContain("[object Object]");

    consoleErrorSpy.mockRestore();
  });
});

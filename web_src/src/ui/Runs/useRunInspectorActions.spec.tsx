import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import type { RunInspectorNodeSection } from "./types";
import { useRunInspectorActions } from "./useRunInspectorActions";

const { cancelRun, showErrorToast, showInfoToast } = vi.hoisted(() => ({
  cancelRun: vi.fn(),
  showErrorToast: vi.fn(),
  showInfoToast: vi.fn(),
}));

vi.mock("@/api-client", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return {
    ...actual,
    canvasesCancelRun: cancelRun,
  };
});

vi.mock("@/lib/toast", () => ({
  showErrorToast,
  showInfoToast,
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
  }: { executionsLoading?: boolean; runOverride?: CanvasesCanvasRun } = {},
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
    vi.clearAllMocks();
    cancelRun.mockResolvedValue({});
  });

  afterEach(() => {
    vi.restoreAllMocks();
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

  it("treats a missing run as no longer active", async () => {
    cancelRun.mockRejectedValue({ code: 5, message: "Not found" });
    const { result } = renderActions([], {
      runOverride: { ...run, id: "run-1" },
    });

    act(() => result.current.stop());

    await waitFor(() => expect(showInfoToast).toHaveBeenCalledWith("Run is no longer active"));
    expect(showErrorToast).not.toHaveBeenCalled();
  });

  it("shows the API message when stopping a run fails", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    cancelRun.mockRejectedValue({ code: 13, message: "Service unavailable" });
    const { result } = renderActions([], {
      runOverride: { ...run, id: "run-1" },
    });

    act(() => result.current.stop());

    await waitFor(() => expect(showErrorToast).toHaveBeenCalledWith("Failed to stop run: Service unavailable"));
    expect(consoleError).toHaveBeenCalledWith("Failed to stop run: Service unavailable");
  });
});

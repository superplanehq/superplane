import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import type { CanvasesCanvasRun, CanvasesCanvasNodeExecution } from "@/api-client";

type DescribeRunQuery = {
  data: { run?: CanvasesCanvasRun } | undefined;
  isError: boolean;
  isLoading: boolean;
};

const { useCanvasRuntimeWebsocketMock, useDescribeRunMock, useEventExecutionsMock } = vi.hoisted(() => ({
  useCanvasRuntimeWebsocketMock: vi.fn(),
  useDescribeRunMock: vi.fn((): DescribeRunQuery => ({ data: undefined, isError: false, isLoading: false })),
  useEventExecutionsMock: vi.fn(
    (): { data?: { executions: CanvasesCanvasNodeExecution[] }; isError: boolean; isLoading: boolean } => ({
      data: undefined,
      isError: false,
      isLoading: false,
    }),
  ),
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: useCanvasRuntimeWebsocketMock,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: undefined, isError: false, isLoading: false }),
  useDescribeRun: useDescribeRunMock,
  useEventExecutions: useEventExecutionsMock,
}));

import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import type { SplitRunPhase } from "./splitRunMocks";

afterEach(() => {
  vi.clearAllMocks();
  useDescribeRunMock.mockReturnValue({ data: undefined, isError: false, isLoading: false });
  useEventExecutionsMock.mockReturnValue({ data: undefined, isError: false, isLoading: false });
});

const PHASE: SplitRunPhase = {
  id: "implement",
  name: "Implement",
  status: "running",
  duration: "1m",
  componentName: "Refund Implementer",
  artifacts: [],
  stream: [],
  canvasSteps: [],
  appId: "app-refund-implementer",
  runId: "run-1",
};

describe("useSplitRunLiveCanvas", () => {
  it.each(["running", "passed", "failed", "cancelled"] as const)(
    "loads individual recorded fleets for %s runs and excludes executions from other runs",
    (status) => {
      useDescribeRunMock.mockReturnValue({
        data: {
          run: {
            id: "run-1",
            rootEvent: { id: "root" },
            executions: [
              { id: "one", componentName: "runnerClaudeCode" },
              { id: "two", componentName: "runnerCodex" },
              { id: "legacy", componentName: "runnerSuperPlane" },
            ],
          },
        },
        isError: false,
        isLoading: false,
      });
      useEventExecutionsMock.mockReturnValue({
        data: {
          executions: [
            { id: "one", runId: "run-1", runnerFleet: "team-one", configuration: { machineType: "current-fleet" } },
            { id: "two", runId: "run-1", runnerFleet: "team-two" },
            { id: "other", runId: "run-2", runnerFleet: "other-fleet" },
          ],
        },
        isError: false,
        isLoading: false,
      });
      const { result } = renderHook(() => useSplitRunLiveCanvas("org-1", { ...PHASE, status }));
      expect(useEventExecutionsMock).toHaveBeenCalledWith(PHASE.appId, "root");
      expect(result.current.runnerActivities?.map((activity) => activity.runnerFleet)).toEqual([
        "team-one",
        "team-two",
        "Unknown",
      ]);
    },
  );
  it("subscribes to the canvas websocket for the live app", () => {
    renderHook(() => useSplitRunLiveCanvas("org-1", PHASE));

    expect(useCanvasRuntimeWebsocketMock).toHaveBeenCalledWith("app-refund-implementer", "org-1", true);
  });

  it("subscribes to the PR feedback canvas when that phase is selected", () => {
    renderHook(() =>
      useSplitRunLiveCanvas("org-1", {
        ...PHASE,
        id: "pr-feedback-run-9",
        appId: "canvas-fb",
        runId: "run-9",
      }),
    );

    expect(useCanvasRuntimeWebsocketMock).toHaveBeenCalledWith("canvas-fb", "org-1", true);
  });

  it("returns the root event id from the describe-run payload", () => {
    useDescribeRunMock.mockReturnValue({
      data: { run: { rootEvent: { id: "event-root" } } },
      isError: false,
      isLoading: false,
    });

    const { result } = renderHook(() => useSplitRunLiveCanvas("org-1", PHASE));

    expect(result.current.rootEventId).toBe("event-root");
  });

  it("keeps the canvas websocket closed when the live app is missing", () => {
    renderHook(() => useSplitRunLiveCanvas("org-1", { ...PHASE, appId: undefined }));

    expect(useCanvasRuntimeWebsocketMock).toHaveBeenCalledWith("", "org-1", false);
  });
});

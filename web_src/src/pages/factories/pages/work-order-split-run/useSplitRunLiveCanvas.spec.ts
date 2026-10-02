import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";

const { useCanvasRuntimeWebsocketMock, useDescribeRunMock } = vi.hoisted(() => ({
  useCanvasRuntimeWebsocketMock: vi.fn(),
  useDescribeRunMock: vi.fn(() => ({ data: undefined, isError: false, isLoading: false })),
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: useCanvasRuntimeWebsocketMock,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: undefined, isError: false, isLoading: false }),
  useDescribeRun: useDescribeRunMock,
}));

import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import type { SplitRunPhase } from "./splitRunMocks";

afterEach(() => {
  vi.clearAllMocks();
  useDescribeRunMock.mockReturnValue({ data: undefined, isError: false, isLoading: false });
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

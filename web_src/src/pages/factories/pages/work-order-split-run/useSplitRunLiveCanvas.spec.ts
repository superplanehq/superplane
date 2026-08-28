import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const { useCanvasRuntimeWebsocketMock } = vi.hoisted(() => ({
  useCanvasRuntimeWebsocketMock: vi.fn(),
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: useCanvasRuntimeWebsocketMock,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({ data: undefined, isError: false, isLoading: false }),
  useDescribeRun: () => ({ data: undefined, isError: false, isLoading: false }),
}));

import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import type { SplitRunPhase } from "./splitRunMocks";

afterEach(() => {
  vi.clearAllMocks();
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

  it("keeps the canvas websocket closed when the live app is missing", () => {
    renderHook(() => useSplitRunLiveCanvas("org-1", { ...PHASE, appId: undefined }));

    expect(useCanvasRuntimeWebsocketMock).toHaveBeenCalledWith("", "org-1", false);
  });
});

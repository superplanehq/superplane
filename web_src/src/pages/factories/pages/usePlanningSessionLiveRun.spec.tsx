import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { runningCreateWithAgentView } from "./createWithAgentDemo";
import { usePlanningSessionLiveRun } from "./usePlanningSessionLiveRun";

const useCanvasRuntimeWebsocket = vi.fn();
const useDescribeRun = vi.fn();

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: (...args: unknown[]) => useCanvasRuntimeWebsocket(...args),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useDescribeRun: (...args: unknown[]) => useDescribeRun(...args),
}));

describe("usePlanningSessionLiveRun", () => {
  it("subscribes to the planning canvas and marks the machine failed when the run fails", () => {
    useDescribeRun.mockReturnValue({ data: { run: { result: "RESULT_FAILED" } } });
    const view = runningCreateWithAgentView();

    const { result } = renderHook(() => usePlanningSessionLiveRun("org-1", view));

    expect(useCanvasRuntimeWebsocket).toHaveBeenCalledWith(view.canvasId, "org-1", true);
    expect(useDescribeRun).toHaveBeenCalledWith(view.canvasId, view.canvasRunId, true);
    expect(result.current.machineStatus).toBe("failed");
  });

  it("stays off until the session has a canvas run", () => {
    useDescribeRun.mockReturnValue({ data: undefined });
    const view = runningCreateWithAgentView({ canvasId: "", canvasRunId: "" });

    renderHook(() => usePlanningSessionLiveRun("org-1", view));

    expect(useCanvasRuntimeWebsocket).toHaveBeenCalledWith("", "org-1", false);
    expect(useDescribeRun).toHaveBeenCalledWith("", null, false);
  });
});

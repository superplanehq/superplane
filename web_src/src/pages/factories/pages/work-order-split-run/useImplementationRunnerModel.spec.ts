import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({
    data: {
      spec: {
        nodes: [{ id: "runner", configuration: { model: "hosted::openrouter::x-ai/grok-4.6" } }],
      },
    },
    isError: false,
    isLoading: false,
  }),
  useDescribeRun: () => ({ data: undefined, isError: false, isLoading: false }),
}));

import { useImplementationRunnerModel } from "./useImplementationRunnerModel";
import type { SplitRunPhase } from "./splitRunMocks";

const IMPLEMENT: SplitRunPhase = {
  id: "implement",
  name: "Implement",
  status: "running",
  duration: "1m",
  componentName: "Implementation",
  artifacts: [],
  stream: [],
  canvasSteps: [],
  canvasKey: "implementation",
  appId: "app-refund-implementer",
  runId: "run-1",
};

describe("useImplementationRunnerModel", () => {
  it("fills Auto from the live implementation canvas", () => {
    const { result } = renderHook(() => useImplementationRunnerModel("org-1", [IMPLEMENT]));

    expect(result.current).toBe("grok-4.6");
  });
});

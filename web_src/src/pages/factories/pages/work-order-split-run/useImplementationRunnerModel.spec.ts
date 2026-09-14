import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

const GROK = { id: "ran", configuration: { model: "hosted::openrouter::x-ai/grok-4.6" } };
const SONNET = { id: "idle", configuration: { model: "anthropic/claude-sonnet-4-6" } };

const { canvasData, runData } = vi.hoisted(() => ({
  canvasData: { spec: { nodes: [] as Array<{ id: string; configuration: { model: string } }> } },
  runData: { run: undefined as { executions?: Array<{ nodeId: string; state: string }> } | undefined },
}));

vi.mock("@/hooks/useCanvasWebsocket", () => ({
  useCanvasRuntimeWebsocket: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvas: () => ({
    data: canvasData,
    isError: false,
    isLoading: false,
  }),
  useDescribeRun: () => ({ data: runData, isError: false, isLoading: false }),
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
    canvasData.spec.nodes = [GROK];
    runData.run = undefined;
    const { result } = renderHook(() => useImplementationRunnerModel("org-1", [IMPLEMENT]));

    expect(result.current).toBe("grok-4.6");
  });

  it("uses the executed runner when the canvas has unused runners", () => {
    canvasData.spec.nodes = [GROK, SONNET];
    runData.run = {
      executions: [{ nodeId: "ran", state: "STATE_STARTED" }],
    };
    const { result } = renderHook(() => useImplementationRunnerModel("org-1", [IMPLEMENT]));

    expect(result.current).toBe("grok-4.6");
  });
});

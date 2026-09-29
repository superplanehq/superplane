import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { useAgentActivityStream } from "../useAgentActivityStream";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { stagesFromFixture } from "./automationsViewModel";
import { LiveAgentSteps } from "./LiveAgentSteps";

vi.mock("../useSplitRunLiveCanvas", () => ({ useSplitRunLiveCanvas: vi.fn() }));
vi.mock("../useAgentActivityStream", () => ({ useAgentActivityStream: vi.fn() }));
vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({ useLiveLogStream: vi.fn() }));

const RUNNING_RUNNER = {
  id: "impl-agent-live",
  at: "12:25:33",
  component: "runnerClaudeCode",
  componentName: "Implementation",
  status: "running" as const,
  executionId: "exec-implementation",
  nodeId: "node-implementation",
};

function implementStage() {
  const stage = stagesFromFixture(SPLIT_RUN_RUNNING).taskStages.find((candidate) => candidate.id === "implement");
  if (!stage) {
    throw new Error("Implement stage is not in the running fixture");
  }
  return stage;
}

function implementPhase() {
  const phase = SPLIT_RUN_RUNNING.phases.find((candidate) => candidate.id === "implement");
  if (!phase) {
    throw new Error("Implement phase is not in the running fixture");
  }
  return phase;
}

describe("LiveAgentSteps", () => {
  beforeEach(() => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [RUNNING_RUNNER],
    });
    vi.mocked(useAgentActivityStream).mockReturnValue({
      activities: [],
      isConnected: true,
      hasConnectedOnce: true,
    });
    vi.mocked(useLiveLogStream).mockReturnValue({
      sections: [],
      orphanLines: [],
      error: null,
      isStreaming: true,
      usageSeries: [],
    } as unknown as ReturnType<typeof useLiveLogStream>);
  });

  it("streams the running run as live activity, not settled note rows", () => {
    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    expect(screen.getByTestId("redesign-live-activity-implement")).toBeInTheDocument();
    expect(vi.mocked(useAgentActivityStream)).toHaveBeenCalledWith(
      expect.objectContaining({
        organizationId: "org-1",
        canvasId: "app-refund-implementer",
        executionId: "exec-implementation",
        active: true,
      }),
    );
    expect(screen.getByTestId("split-run-intent-thinking")).toHaveTextContent("Starting agent…");
  });

  it("keeps the finished run on the settled step list", () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    const stage = { ...implementStage(), status: "passed" as const };

    render(<LiveAgentSteps stage={stage} phase={implementPhase()} organizationId="org-1" />);

    expect(screen.queryByTestId("redesign-live-activity-implement")).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-agent-steps-implement")).toBeInTheDocument();
  });
});

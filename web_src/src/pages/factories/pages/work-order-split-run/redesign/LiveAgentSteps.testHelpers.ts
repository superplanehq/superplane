import { emptyAgentActivityState, type AgentActivity } from "../agentActivity";
import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import type { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { stagesFromFixture } from "./automationsViewModel";

export const RUNNING_RUNNER = {
  id: "impl-agent-live",
  at: "12:25:33",
  component: "runnerClaudeCode",
  componentName: "Implementation",
  status: "running" as const,
  executionId: "exec-implementation",
  nodeId: "node-implementation",
};

export const LIVE_ACTIVITY: AgentActivity = {
  id: "activity-1",
  provider: "opencode",
  status: "running",
  sequence: 4,
  truncated: false,
  items: [
    {
      type: "content",
      id: "reasoning-1",
      kind: "reasoning",
      text: "Inspecting the retry path.",
      status: "running",
      truncated: false,
    },
  ],
};

export function implementStage() {
  const stage = stagesFromFixture(SPLIT_RUN_RUNNING).taskStages.find((candidate) => candidate.id === "implement");
  if (!stage) {
    throw new Error("Implement stage is not in the running fixture");
  }
  return stage;
}

export function implementPhase() {
  const phase = SPLIT_RUN_RUNNING.phases.find((candidate) => candidate.id === "implement");
  if (!phase) {
    throw new Error("Implement phase is not in the running fixture");
  }
  return phase;
}

export function idleStream(overrides: Record<string, unknown> = {}) {
  return {
    sections: [],
    orphanLines: [],
    error: null,
    isStreaming: true,
    isLoading: false,
    usageSeries: [],
    activityState: emptyAgentActivityState,
    ...overrides,
  } as unknown as ReturnType<typeof useLiveLogStream>;
}

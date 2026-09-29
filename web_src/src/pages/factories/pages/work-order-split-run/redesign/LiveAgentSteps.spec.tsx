import { emptyAgentActivityState, type AgentActivity } from "../agentActivity";
import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SPLIT_RUN_RUNNING } from "../splitRunMocks";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { stagesFromFixture } from "./automationsViewModel";
import { LiveAgentSteps } from "./LiveAgentSteps";

vi.mock("../useSplitRunLiveCanvas", () => ({ useSplitRunLiveCanvas: vi.fn() }));
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

const LIVE_ACTIVITY: AgentActivity = {
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

function idleStream(overrides: Record<string, unknown> = {}) {
  return {
    sections: [],
    orphanLines: [],
    error: null,
    isStreaming: true,
    usageSeries: [],
    activityState: emptyAgentActivityState,
    ...overrides,
  } as unknown as ReturnType<typeof useLiveLogStream>;
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
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream());
  });

  it("streams the running turn like refinement chat, not expanded tool dumps", async () => {
    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        sections: [
          {
            index: 1,
            text: "Implementation",
            kind: "prompt",
            preview: "",
            lines: [],
            events: [
              {
                kind: "tools",
                id: "tools-1",
                tools: [
                  {
                    id: "tool-1",
                    kind: "read",
                    text: "/home/node/.superplane/homes/repo/web_src/src/hooks/useFactoryPRFeedbackData.ts",
                    lines: [
                      "<path>/home/node/.superplane/homes/repo/web_src/src/hooks/useFactoryPRFeedbackData.ts</path>",
                    ],
                    status: "passed",
                    duration_ms: 12,
                  },
                ],
              },
            ],
            activities: [LIVE_ACTIVITY],
            status: "running",
            duration_ms: null,
            started_at: 1,
            collapsed: false,
          },
        ],
        activityState: { activities: [LIVE_ACTIVITY], seenEventIds: new Set() },
      }),
    );

    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    await waitFor(() => {
      expect(screen.getByTestId("redesign-agent-steps-implement")).toBeInTheDocument();
    });
    expect(screen.getByTestId("redesign-live-activity-implement")).toBeInTheDocument();
    expect(screen.getByText("Inspecting the retry path.")).toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Thinking" })).toBeInTheDocument();
    expect(
      screen.queryByText("/home/node/.superplane/homes/repo/web_src/src/hooks/useFactoryPRFeedbackData.ts"),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("<path>")).not.toBeInTheDocument();
  });

  it("maps OpenCode section events into the live chat when activity records are absent", async () => {
    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        sections: [
          {
            index: 1,
            text: "Implementation",
            kind: "prompt",
            preview: "",
            lines: [],
            events: [{ kind: "note", text: "Let me read the factory handler." }],
            status: "running",
            duration_ms: null,
            started_at: 1,
            collapsed: false,
          },
        ],
      }),
    );

    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    await waitFor(() => {
      expect(screen.getByText("Let me read the factory handler.")).toBeInTheDocument();
    });
    expect(screen.getByTestId("redesign-agent-steps-implement")).toBeInTheDocument();
    expect(screen.getByTestId("redesign-live-activity-implement")).toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Planning next step…" })).toBeInTheDocument();
    expect(screen.queryByText("Still working")).not.toBeInTheDocument();
  });

  it("keeps the finished run on the settled step list when no transcript arrives", () => {
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

  it("keeps finished command sections as collapsible step rows", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        isStreaming: false,
        sections: [
          {
            index: 1,
            text: "Clone Repo",
            kind: "bash",
            preview: "cd /tmp && git clone https://example.com/repo.git",
            lines: [],
            events: [],
            status: "passed",
            duration_ms: 20,
            started_at: 1,
            collapsed: true,
          },
          {
            index: 2,
            text: "Implementation",
            kind: "prompt",
            preview: "",
            lines: [],
            events: [{ kind: "note", text: "I will download the bun zip and extract it." }],
            status: "passed",
            duration_ms: 40,
            started_at: 2,
            collapsed: true,
          },
        ],
      }),
    );
    const stage = { ...implementStage(), status: "passed" as const };

    render(<LiveAgentSteps stage={stage} phase={implementPhase()} organizationId="org-1" />);

    await waitFor(() => {
      expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    });
    expect(screen.getByText("Implementation")).toBeInTheDocument();
    expect(screen.queryByText("Bash")).not.toBeInTheDocument();
    expect(screen.queryByText("Prompt")).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-agent-steps-implement")).toBeInTheDocument();
    expect(screen.queryByTestId("redesign-live-activity-implement")).not.toBeInTheDocument();
    expect(screen.queryByText("I will download the bun zip and extract it.")).not.toBeInTheDocument();
  });
});

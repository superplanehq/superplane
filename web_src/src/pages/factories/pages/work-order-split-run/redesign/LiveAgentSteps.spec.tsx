import { LOADING_REVEAL_CLASSNAME } from "../../../lib/loadingReveal";
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

  it("keeps earlier recorded turns in the open running step", async () => {
    const prior: AgentActivity = {
      id: "activity-0",
      provider: "opencode",
      status: "passed",
      sequence: 2,
      truncated: false,
      items: [
        {
          type: "content",
          id: "note-0",
          kind: "assistant",
          text: "First I inspect the host.",
          status: "passed",
          truncated: false,
        },
      ],
    };
    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        sections: [
          {
            index: 1,
            text: "Implementation",
            kind: "prompt",
            preview: "",
            lines: [],
            events: [],
            activities: [prior, LIVE_ACTIVITY],
            status: "running",
            duration_ms: null,
            started_at: 1,
            collapsed: false,
          },
        ],
        activityState: { activities: [prior, LIVE_ACTIVITY], seenEventIds: new Set() },
      }),
    );

    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    await waitFor(() => {
      expect(screen.getByText("First I inspect the host.")).toBeInTheDocument();
    });
    expect(screen.getByText("Inspecting the retry path.")).toBeInTheDocument();
  });

  it("does not put another runner transcript under the open step", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [
        { ...RUNNING_RUNNER, id: "other-runner", executionId: "exec-other", nodeId: "node-other", status: "passed" },
        RUNNING_RUNNER,
      ],
    });
    const other: AgentActivity = {
      id: "activity-other",
      provider: "opencode",
      status: "passed",
      sequence: 1,
      truncated: false,
      items: [
        {
          type: "content",
          id: "note-other",
          kind: "assistant",
          text: "Clone finished on the other runner.",
          status: "passed",
          truncated: false,
        },
      ],
    };
    vi.mocked(useLiveLogStream).mockImplementation((executionId: string) => {
      if (executionId === "exec-other") {
        return idleStream({
          isStreaming: false,
          activityState: { activities: [other], seenEventIds: new Set() },
        });
      }
      return idleStream({
        activityState: { activities: [LIVE_ACTIVITY], seenEventIds: new Set() },
      });
    });

    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    await waitFor(() => {
      expect(screen.getByText("Inspecting the retry path.")).toBeInTheDocument();
    });
    expect(screen.queryByText("Clone finished on the other runner.")).not.toBeInTheDocument();
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

  it("keeps the finished run on the settled step list when no transcript arrives", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    const stage = {
      ...implementStage(),
      status: "passed" as const,
      agentSteps: [
        {
          id: "settled-implementation",
          title: "Implementation",
          type: "prompt" as const,
          status: "passed" as const,
          summary: "",
          toolCount: 0,
          events: [],
        },
      ],
    };

    render(<LiveAgentSteps stage={stage} phase={implementPhase()} organizationId="org-1" />);

    expect(await screen.findByText("Implementation")).toBeInTheDocument();
    expect(screen.queryByTestId("redesign-live-activity-implement")).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-agent-steps-implement")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
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

  it("shows a text skeleton while a live run has no log lines", async () => {
    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    const skeleton = await screen.findByRole("status", { name: "Waiting for logs" });
    expect(skeleton).toHaveAttribute("aria-busy", "true");
    expect(skeleton.querySelectorAll(".animate-pulse")).toHaveLength(4);
    expect(skeleton.querySelector(".w-full")).toBeTruthy();
    expect(skeleton.querySelector(".w-11\\/12")).toBeTruthy();
    expect(skeleton.querySelector(".w-4\\/5")).toBeTruthy();
    expect(skeleton.querySelector(".w-2\\/3")).toBeTruthy();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
    expect(screen.queryByTestId("redesign-agent-steps-implement")).not.toBeInTheDocument();
  });

  it("replaces the log skeleton with the real line", async () => {
    const view = render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);
    await screen.findByRole("status", { name: "Waiting for logs" });

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
    view.rerender(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    expect(await screen.findByText("Let me read the factory handler.")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.getByTestId("redesign-agent-steps-implement").parentElement).toHaveClass(
      ...LOADING_REVEAL_CLASSNAME.split(" "),
    );
  });

  it("keeps another runner's lines and hides the log skeleton", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [
        { ...RUNNING_RUNNER, id: "other-runner", executionId: "exec-other", nodeId: "node-other", status: "passed" },
        RUNNING_RUNNER,
      ],
    });
    vi.mocked(useLiveLogStream).mockImplementation((executionId: string) => {
      if (executionId === "exec-other") {
        return idleStream({
          isStreaming: false,
          sections: [
            {
              index: 1,
              text: "Clone Repo",
              kind: "bash",
              preview: "git clone",
              lines: [],
              events: [],
              status: "passed",
              duration_ms: 20,
              started_at: 1,
              collapsed: true,
            },
          ],
        });
      }
      return idleStream();
    });

    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    expect(await screen.findByText("Clone Repo")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
  });

  it("keeps the log fetch error as visible text", async () => {
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ error: "timeout", isStreaming: false }));

    render(<LiveAgentSteps stage={implementStage()} phase={implementPhase()} organizationId="org-1" />);

    expect(await screen.findByText("Something went wrong while fetching logs.")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
  });

  it("holds the log skeleton until a finished transcript fetch settles", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        isStreaming: true,
        sections: [
          {
            index: 1,
            text: "Implementation",
            kind: "prompt",
            preview: "",
            lines: [],
            events: [{ kind: "note", text: "I will download the bun zip and extract it." }],
            status: "passed",
            duration_ms: 40,
            started_at: 2,
            collapsed: false,
          },
        ],
      }),
    );
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    const view = render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(screen.getByRole("status", { name: "Waiting for logs" })).toBeInTheDocument();
    expect(screen.queryByText("Implementation")).not.toBeInTheDocument();
    expect(screen.queryByText("I will download the bun zip and extract it.")).not.toBeInTheDocument();
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();

    vi.mocked(useLiveLogStream).mockReturnValue(
      idleStream({
        isStreaming: false,
        sections: [
          {
            index: 1,
            text: "Clone Repo",
            kind: "bash",
            preview: "git clone",
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
    view.rerender(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("Clone Repo")).toBeInTheDocument();
    expect(screen.getByText("Implementation")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.queryByText("I will download the bun zip and extract it.")).not.toBeInTheDocument();
  });

  it("shows the log skeleton while a finished run still fetches its transcript", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: true }));
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(screen.getByRole("status", { name: "Waiting for logs" })).toBeInTheDocument();
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
  });

  it("shows the log skeleton while the finished run canvas is still loading", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: true,
      canvas: undefined,
      stream: [],
    });
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByRole("status", { name: "Waiting for logs" })).toBeInTheDocument();
    expect(screen.queryByText("No steps for this run.")).not.toBeInTheDocument();
  });

  it("keeps the empty sentence for a finished run with no steps", async () => {
    vi.mocked(useSplitRunLiveCanvas).mockReturnValue({
      enabled: true,
      isError: false,
      isLoading: false,
      canvas: undefined,
      stream: [{ ...RUNNING_RUNNER, status: "passed" as const }],
    });
    vi.mocked(useLiveLogStream).mockReturnValue(idleStream({ isStreaming: false }));
    const stage = { ...implementStage(), status: "passed" as const, agentSteps: [] };

    render(
      <LiveAgentSteps
        stage={stage}
        phase={implementPhase()}
        organizationId="org-1"
        emptyNote="No steps for this run."
      />,
    );

    expect(await screen.findByText("No steps for this run.")).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
  });

  it("keeps Starting agent when the live stream has not reported a waiting note", () => {
    const stage = { ...implementStage(), agentSteps: [] };
    const phase = { ...implementPhase(), appId: undefined };

    render(<LiveAgentSteps stage={stage} phase={phase} organizationId="org-1" />);

    expect(screen.getByRole("status", { name: "Starting agent…" })).toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "Waiting for logs" })).not.toBeInTheDocument();
  });
});

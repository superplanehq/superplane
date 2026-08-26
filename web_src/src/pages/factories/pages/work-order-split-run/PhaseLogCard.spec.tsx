import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { groupClaudeSteps, PhaseLogCard, toolCallSummary } from "./PhaseLogCard";
import type { SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";

const useLiveLogStreamMock = vi.fn();

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue({
    sections: [],
    orphanLines: [],
    error: null,
    isStreaming: false,
    toggleSection: vi.fn(),
    scrollRef: { current: null },
  });
});

const PHASE: SplitRunPhase = {
  id: "plan",
  name: "Plan",
  status: "passed",
  duration: "1m",
  componentName: "Planning",
  artifacts: [],
  stream: [],
  canvasSteps: [],
};

const LONG_NOTE =
  "Now let me check factories.proto Delete rpc absence explicitly and PermissionTooltip component quickly, plus check showSuccessToast import paths.";

function line(
  partial: Partial<SplitRunStreamLine> & Pick<SplitRunStreamLine, "id" | "componentName">,
): SplitRunStreamLine {
  return {
    at: "12:00",
    status: "passed",
    nodeId: "planner-agent",
    ...partial,
  };
}

const PLANNING_STREAM: SplitRunStreamLine[] = [
  line({ id: "planner-agent", componentName: "Agent - Plan for GH Issue", componentType: "Run Claude Code" }),
  line({
    id: "step-clone",
    note: true,
    componentName: "Clone Repo",
    componentType: "bash",
    status: "passed",
    detail: "Cloning into 'superplane'...",
  }),
  line({
    id: "step-fail",
    note: true,
    componentName: "Run Tests",
    componentType: "bash",
    status: "failed",
    detail: "FAIL pkg/foo",
  }),
  line({
    id: "step-write",
    note: true,
    componentName: "Write Implementation Plan",
    componentType: "prompt",
  }),
  line({
    id: "cmd-cat",
    note: true,
    noteParentId: "step-write",
    noteDepth: 1,
    componentName: "cat /tmp/ORDER.md",
    componentType: "bash",
    detail: "## Goal\nAdd a menu.",
  }),
  line({
    id: "cmd-note",
    note: true,
    noteParentId: "step-write",
    noteDepth: 1,
    componentName: LONG_NOTE,
    componentType: "note",
  }),
  line({
    id: "cmd-read",
    note: true,
    noteParentId: "step-write",
    noteDepth: 1,
    componentName: "LineListCard.tsx",
    componentType: "read",
  }),
  line({ id: "step-out", note: true, componentName: "Use plan as output", componentType: "bash" }),
];

describe("toolCallSummary", () => {
  it("names read files and ran commands", () => {
    expect(toolCallSummary([{ type: "read" }, { type: "read" }, { type: "bash" }])).toBe("Read 2 files, ran 1 command");
    expect(toolCallSummary([{ componentType: "read" }, { componentType: "bash" }])).toBe("Read 1 file, ran 1 command");
    expect(toolCallSummary([{ type: "read" }])).toBe("Read 1 file");
    expect(toolCallSummary([{ type: "bash" }, { type: "bash" }])).toBe("Ran 2 commands");
  });
});

describe("groupClaudeSteps", () => {
  it("keeps tools and agent notes in log order", () => {
    const write = groupClaudeSteps(PLANNING_STREAM.filter((entry) => entry.note)).find(
      (step) => step.line.id === "step-write",
    );
    expect(write?.events.map((event) => event.kind)).toEqual(["tools", "note", "tools"]);
    expect(write?.events[0]).toMatchObject({
      kind: "tools",
      tools: [{ componentName: "cat /tmp/ORDER.md" }],
    });
    expect(write?.events[1]).toMatchObject({
      kind: "note",
      line: { componentName: LONG_NOTE },
    });
    expect(write?.events[2]).toMatchObject({
      kind: "tools",
      tools: [{ componentName: "LineListCard.tsx" }],
    });
  });
});

describe("PhaseLogCard collapsed stream", () => {
  it("hides node children until the node expands", async () => {
    const user = userEvent.setup();
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    expect(screen.getByText("Agent - Plan for GH Issue")).toBeInTheDocument();
    expect(screen.queryByText("Clone Repo")).not.toBeInTheDocument();
    expect(screen.queryByText(LONG_NOTE)).not.toBeInTheDocument();

    await user.click(screen.getByText("Agent - Plan for GH Issue"));

    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    expect(screen.getByText("Write Implementation Plan")).toBeInTheDocument();
    expect(screen.getByText("Use plan as output")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-line-step-clone")).getByText("✓")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-line-step-fail")).getByText("✗")).toBeInTheDocument();
    expect(screen.queryByText("Cloning into 'superplane'...")).not.toBeInTheDocument();
    expect(screen.queryByText(LONG_NOTE)).not.toBeInTheDocument();
    expect(screen.queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();
  });

  it("expands the selected node in the log", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} selectedNodeId="planner-agent" />);

    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    expect(screen.getByText("Write Implementation Plan")).toBeInTheDocument();
    expect(screen.queryByText(LONG_NOTE)).not.toBeInTheDocument();
  });

  it("maps live log sections under an expanded runner node", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        {
          index: 0,
          text: "Prepare Claude Code",
          kind: "setup",
          preview: "npm install",
          lines: [],
          events: [],
          status: "passed",
          duration_ms: 10,
          started_at: 1,
          collapsed: true,
        },
        {
          index: 1,
          text: "Set Up Git User",
          kind: "bash",
          preview: 'echo "Using superplaneagent@superplane.com"',
          lines: ["Using superplaneagent@superplane.com"],
          events: [],
          status: "passed",
          duration_ms: 20,
          started_at: 1,
          collapsed: true,
        },
        {
          index: 5,
          text: "Implementation",
          kind: "prompt",
          preview: "You are implementing a fix",
          lines: [],
          events: [
            { kind: "note", text: "Gathering issue context first." },
            {
              kind: "tools",
              id: "5-tools-0",
              tools: [
                {
                  id: "5-tool-0",
                  kind: "read",
                  text: "pkg/foo.go",
                  lines: ["package workers"],
                  status: "passed",
                  duration_ms: 80,
                },
              ],
            },
          ],
          status: "passed",
          duration_ms: 900,
          started_at: 1,
          collapsed: true,
        },
      ],
      orphanLines: [],
      error: null,
      isStreaming: false,
      toggleSection: vi.fn(),
      scrollRef: { current: null },
    });

    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "runner-agent",
            nodeId: "runner-agent",
            componentName: "Run Claude Code",
            componentType: "Run Claude Code",
            component: "runnerClaudeCode",
            executionId: "exec-1",
          }),
        ]}
      />,
    );

    await user.click(screen.getByTestId("split-run-node-toggle-runner-agent"));

    expect(screen.queryByText("Prepare Claude Code")).not.toBeInTheDocument();
    expect(screen.getByText("bash")).toBeInTheDocument();
    expect(screen.getByText('echo "Using superplaneagent@superplane.com"')).toBeInTheDocument();
    expect(screen.getByText("prompt")).toBeInTheDocument();
    expect(screen.getByText("You are implementing a fix")).toBeInTheDocument();

    await user.click(screen.getByText("You are implementing a fix"));
    expect(screen.getByText("Gathering issue context first.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Read 1 file" })).toBeInTheDocument();
  });

  it("keeps yaml notes when live sections are only setup", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        {
          index: 0,
          text: "Prepare Claude Code",
          kind: "setup",
          preview: "npm install",
          lines: [],
          events: [],
          status: "passed",
          duration_ms: 10,
          started_at: 1,
          collapsed: true,
        },
      ],
      orphanLines: [],
      error: null,
      isStreaming: false,
      toggleSection: vi.fn(),
      scrollRef: { current: null },
    });

    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "planner-agent",
            nodeId: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            componentType: "Run Claude Code",
            component: "runnerClaudeCode",
            executionId: "exec-1",
          }),
          ...PLANNING_STREAM.filter((row) => row.id !== "planner-agent"),
        ]}
      />,
    );

    await user.click(screen.getByTestId("split-run-node-toggle-planner-agent"));
    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    expect(screen.queryByText("Prepare Claude Code")).not.toBeInTheDocument();
  });

  it("shows a live log error on an expanded runner node", async () => {
    const user = userEvent.setup();
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: [],
      error: "boom",
      isStreaming: false,
      toggleSection: vi.fn(),
      scrollRef: { current: null },
    });

    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        organizationId="org-1"
        canvasId="canvas-1"
        stream={[
          line({
            id: "runner-agent",
            nodeId: "runner-agent",
            componentName: "Run Claude Code",
            componentType: "Run Claude Code",
            component: "runnerClaudeCode",
            executionId: "exec-1",
          }),
        ]}
      />,
    );

    await user.click(screen.getByTestId("split-run-node-toggle-runner-agent"));
    expect(screen.getByText("Something went wrong while fetching logs.")).toBeInTheDocument();
  });

  it("expands bash output from the step line", async () => {
    const user = userEvent.setup();
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    await user.click(screen.getByText("Agent - Plan for GH Issue"));
    await user.click(screen.getByText("Clone Repo"));

    expect(screen.getByText("Cloning into 'superplane'...")).toBeInTheDocument();
    await user.click(screen.getByText("Run Tests"));
    expect(screen.getByText("FAIL pkg/foo")).toBeInTheDocument();
  });

  it("shows agent text and a collapsed tool summary when the prompt expands", async () => {
    const user = userEvent.setup();
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    await user.click(screen.getByText("Agent - Plan for GH Issue"));
    await user.click(screen.getByText("Write Implementation Plan"));

    const note = screen.getByText(LONG_NOTE);
    expect(note).toBeInTheDocument();
    expect(note).not.toHaveClass("truncate");
    expect(note).toHaveClass("whitespace-normal");
    expect(screen.getByRole("button", { name: "Ran 1 command" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Read 1 file" })).toBeInTheDocument();
    expect(screen.queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();
    expect(screen.queryByText("LineListCard.tsx")).not.toBeInTheDocument();
    expect(screen.queryByText("note")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ran 1 command" }));

    const stream = screen.getByTestId("split-run-stream-plan");
    expect(within(stream).getByText("cat /tmp/ORDER.md")).toBeInTheDocument();
    expect(within(stream).queryByText(/## Goal/)).not.toBeInTheDocument();
    expect(within(stream).queryByText("LineListCard.tsx")).not.toBeInTheDocument();

    await user.click(within(stream).getByText("cat /tmp/ORDER.md"));
    expect(within(stream).getByText(/## Goal/)).toBeInTheDocument();
    expect(within(stream).getByText(/Add a menu/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Read 1 file" }));
    expect(within(stream).getByText("LineListCard.tsx")).toBeInTheDocument();
  });

  it("shows a check pill on the phase title", () => {
    render(
      <PhaseLogCard
        phase={{
          ...PHASE,
          id: "score",
          name: "Score",
          componentName: "Score",
          checks: [
            {
              id: "wo-review-pay-842-confidence",
              name: "Confidence score",
              score: 5,
              maxScore: 5,
              format: "fraction",
              level: "positive",
              summary: "Analysis complete.",
              sourceName: "Score",
            },
          ],
        }}
        expanded
      />,
    );

    const score = screen.getByTestId("split-run-phase-score");
    expect(within(score).getByTestId("split-run-check-wo-review-pay-842-confidence")).toHaveTextContent("5/5");
  });
});

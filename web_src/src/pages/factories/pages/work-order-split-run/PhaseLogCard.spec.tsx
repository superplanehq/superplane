import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { groupClaudeSteps, PhaseLogCard, toolCallSummary } from "./PhaseLogCard";
import { idleLiveLogStream, line, LONG_NOTE, PHASE, PLANNING_STREAM } from "./PhaseLogCard.testHelpers";

const useLiveLogStreamMock = vi.fn();

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
});

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

  it("does not let blank notes split consecutive tools", () => {
    const grouped = groupClaudeSteps([
      line({ id: "step-write", note: true, componentType: "prompt", componentName: "Plan" }),
      line({
        id: "t1",
        note: true,
        noteParentId: "step-write",
        componentType: "bash",
        componentName: "echo a",
      }),
      line({
        id: "blank",
        note: true,
        noteParentId: "step-write",
        componentType: "note",
        componentName: "  ",
      }),
      line({
        id: "t2",
        note: true,
        noteParentId: "step-write",
        componentType: "bash",
        componentName: "echo b",
      }),
    ]);

    expect(grouped[0]?.events).toHaveLength(1);
    expect(grouped[0]?.events[0]).toMatchObject({
      kind: "tools",
      tools: [{ id: "t1" }, { id: "t2" }],
    });
  });
});

describe("PhaseLogCard collapsed stream", () => {
  it("shows node steps without a caret and keeps them open", async () => {
    const user = userEvent.setup();
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    const node = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(node.querySelector(".lucide-chevron-right")).toBeNull();
    expect(node.className).toMatch(/\bbg-muted\b/);
    expect(node.className).not.toMatch(/\bbg-background\b/);
    expect(node.className).not.toMatch(/border-b/);
    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    expect(screen.getByText("Write Implementation Plan")).toBeInTheDocument();
    expect(screen.getByText("Use plan as output")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-line-step-clone")).getByText("✓")).toBeInTheDocument();
    expect(within(screen.getByTestId("split-run-stream-line-step-fail")).getByText("✗")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-stream-line-step-clone").querySelector(".lucide-chevron-right")).toBeNull();
    expect(screen.getByTestId("split-run-stream-line-step-clone").className).toMatch(/\bbg-muted\b/);
    expect(screen.getByTestId("split-run-stream-line-step-clone").className).not.toMatch(/\bbg-background\b/);
    expect(screen.getByText("Cloning into 'superplane'...")).toBeInTheDocument();
    expect(screen.getByText(LONG_NOTE)).toBeInTheDocument();
    expect(screen.queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();

    await user.click(screen.getByText("Agent - Plan for GH Issue"));
    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    await user.click(screen.getByText("Clone Repo"));
    expect(screen.getByText("Cloning into 'superplane'...")).toBeInTheDocument();
  });

  it("wraps long bash and prompt titles", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    const bash = screen.getByTestId("split-run-stream-line-step-clone");
    const bashTitle = within(bash).getByText("Clone Repo");
    expect(bash).not.toHaveClass("whitespace-nowrap");
    expect(bashTitle).not.toHaveClass("truncate");
    expect(bashTitle).toHaveClass("whitespace-normal", "break-words");

    const prompt = screen.getByTestId("split-run-stream-line-step-write");
    const promptTitle = within(prompt).getByText("Write Implementation Plan");
    expect(prompt).not.toHaveClass("whitespace-nowrap");
    expect(promptTitle).not.toHaveClass("truncate");
    expect(promptTitle).toHaveClass("whitespace-normal", "break-words");

    const output = within(screen.getByTestId("split-run-stream-line-step-clone").parentElement as HTMLElement)
      .getByTestId("split-run-stream-output")
      .querySelector("pre");
    expect(output).toHaveClass("whitespace-pre-wrap", "break-words");
    expect(output).not.toHaveClass("truncate");
  });

  it("expands the selected node in the log", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} selectedNodeId="planner-agent" />);

    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    expect(screen.getByText("Write Implementation Plan")).toBeInTheDocument();
    expect(screen.getByText(LONG_NOTE)).toBeInTheDocument();
  });

  it("maps live log sections under an expanded runner node", () => {
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

    expect(screen.queryByText("Prepare Claude Code")).not.toBeInTheDocument();
    expect(screen.getByText("bash")).toBeInTheDocument();
    expect(screen.getByText('echo "Using superplaneagent@superplane.com"')).toBeInTheDocument();
    expect(screen.getByText("prompt")).toBeInTheDocument();
    expect(screen.getByText("You are implementing a fix")).toBeInTheDocument();
    expect(screen.getByText("Gathering issue context first.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Read 1 file" })).toBeInTheDocument();
  });

  it("collapses session setup and bash when compactSessionLog is on", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        {
          index: 1,
          text: "Clone Repo",
          kind: "bash",
          preview: "git clone repo",
          lines: ["Cloning into 'repo'..."],
          events: [],
          status: "passed",
          duration_ms: 20,
          started_at: 1,
          collapsed: true,
        },
        {
          index: 5,
          text: "Plan with the user",
          kind: "prompt",
          preview: "You are in a SuperPlane planning session",
          lines: [],
          events: [
            { kind: "note", text: "Planning session tools enabled" },
            { kind: "note", text: "permission mode: bypassPermissions" },
            { kind: "note", text: "The repository is ready. What do you want to do?" },
            { kind: "note", text: '{"message":"Hi! I am ready to help you plan work in this repository."}' },
            {
              kind: "tools",
              id: "5-tools-0",
              tools: [
                {
                  id: "5-tool-0",
                  kind: "read",
                  text: "README.md",
                  lines: ["# Store"],
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
        compactSessionLog
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

    expect(screen.queryByText("git clone repo")).not.toBeInTheDocument();
    expect(screen.queryByText("Planning session tools enabled")).not.toBeInTheDocument();
    expect(screen.queryByText("permission mode: bypassPermissions")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ran 1 command" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Setup" })).not.toBeInTheDocument();
    expect(screen.queryByText("You are in a SuperPlane planning session")).not.toBeInTheDocument();
    expect(screen.queryByText("prompt")).not.toBeInTheDocument();
    expect(screen.queryByText(/"message":"Hi! I am ready/)).not.toBeInTheDocument();
    expect(screen.getByText("The repository is ready. What do you want to do?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Read 1 file, ran 1 command" })).toBeInTheDocument();
  });

  it("marks user replies in the compact session log", () => {
    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        compactSessionLog
        stream={[
          line({
            id: "runner-agent",
            nodeId: "runner-agent",
            componentName: "Agent",
            componentType: "Run Claude Code",
            component: "runnerClaudeCode",
          }),
          line({
            id: "user-1",
            nodeId: "runner-agent",
            note: true,
            componentType: "prompt",
            componentName: "Add a Size field",
          }),
          line({
            id: "agent-1",
            nodeId: "runner-agent",
            note: true,
            componentType: "note",
            componentName: "I can draft that.",
          }),
        ]}
      />,
    );

    const userNote = screen.getByTestId("split-run-user-note");
    expect(userNote).toHaveTextContent("You");
    expect(userNote).toHaveTextContent("Add a Size field");
    expect(screen.getByText("I can draft that.")).toBeInTheDocument();
    expect(screen.getByText("I can draft that.").closest("[data-testid='split-run-user-note']")).toBeNull();
  });
});

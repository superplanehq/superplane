import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, LONG_NOTE, PHASE, PLANNING_STREAM } from "./PhaseLogCard.testHelpers";

const useLiveLogStreamMock = vi.fn();

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
});

describe("PhaseLogCard stream details", () => {
  it("labels a survey reply as You (survey response)", () => {
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
            id: "user-survey",
            nodeId: "runner-agent",
            note: true,
            componentType: "prompt",
            userTalk: "survey",
            componentName: "What is the priority? High",
          }),
        ]}
      />,
    );

    const userNote = screen.getByTestId("split-run-user-note");
    expect(userNote).toHaveTextContent("You (survey response)");
    expect(userNote).toHaveTextContent("What is the priority? High");
  });

  it("shows the submitted survey answer, not the live log's own wording, in the transcript", () => {
    // The live runner log recognizes the turn as the user's survey reply
    // (so it should be labeled "You (survey response)"), but its own text
    // for that turn is a summary that buries the chosen answer behind the
    // word "skipped". The transcript must still show the answer the user
    // actually picked.
    useLiveLogStreamMock.mockReturnValue({
      sections: [
        {
          index: 0,
          text: "What is the priority? High",
          kind: "prompt",
          preview: "What is the priority? High (rest marked as skipped)",
          lines: [],
          events: [],
          status: "passed",
          duration_ms: 10,
          started_at: 1,
          collapsed: false,
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
            componentName: "Agent",
            componentType: "Run Claude Code",
            component: "runnerClaudeCode",
            executionId: "exec-1",
            status: "running",
          }),
          line({
            id: "user-survey",
            nodeId: "runner-agent",
            note: true,
            componentType: "prompt",
            userTalk: "survey",
            componentName: "What is the priority? High",
          }),
        ]}
      />,
    );

    const userNote = screen.getByTestId("split-run-user-note");
    expect(userNote).toHaveTextContent("You (survey response)");
    expect(userNote).toHaveTextContent("What is the priority? High");
    expect(userNote).not.toHaveTextContent("skipped");
  });

  it("keeps yaml notes when live sections are only setup", () => {
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

    expect(screen.getByText("Clone Repo")).toBeInTheDocument();
    expect(screen.queryByText("Prepare Claude Code")).not.toBeInTheDocument();
  });

  it("shows orphan live-log lines instead of Waiting for logs", () => {
    useLiveLogStreamMock.mockReturnValue({
      sections: [],
      orphanLines: ["Claude Code ready", "Cloning into 'repo'..."],
      error: null,
      isStreaming: true,
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
            status: "running",
          }),
        ]}
      />,
    );

    expect(screen.getByText("Claude Code ready")).toBeInTheDocument();
    expect(screen.getByText("Cloning into 'repo'...")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
  });

  it("shows a live log error on an expanded runner node", () => {
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

    expect(screen.getByText("Something went wrong while fetching logs.")).toBeInTheDocument();
  });

  it("shows bash output on the step line", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    expect(screen.getByText("Cloning into 'superplane'...")).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-output-indent")).not.toBeInTheDocument();
    expect(screen.getByText("FAIL pkg/foo")).toBeInTheDocument();
  });

  it("does not show line numbers", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    expect(screen.queryAllByTestId("split-run-log-line-no")).toHaveLength(0);
  });

  it("tints collapsed phases on hover and leaves log lines clear", () => {
    const { rerender } = render(<PhaseLogCard phase={PHASE} expanded={false} />);

    expect(screen.getByTestId("split-run-phase-plan").firstElementChild?.className).toMatch(/hover:bg-/);

    rerender(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    expect(screen.getByTestId("split-run-phase-plan").firstElementChild?.className).not.toMatch(/hover:bg-/);

    const node = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(node.className).not.toMatch(/hover:bg-/);
    const step = screen.getByTestId("split-run-stream-line-step-clone");
    expect(step.className).not.toMatch(/hover:bg-/);

    const outputLine = screen.getAllByTestId("split-run-stream-output")[0]?.firstElementChild;
    expect(outputLine?.className).not.toMatch(/hover:bg-/);

    expect(screen.getByText(LONG_NOTE).closest("[data-testid^='split-run-stream-line-']")?.className).not.toMatch(
      /hover:bg-/,
    );
    expect(screen.getByRole("button", { name: "Ran 1 command" }).className).not.toMatch(/hover:bg-/);
  });

  it("pins the open phase, node, and step while their output scrolls", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    expect(screen.getByTestId("split-run-stream-plan").className).not.toMatch(/overflow-hidden/);

    const phase = screen.getByTestId("split-run-automation-header-plan");
    expect(phase.className).toMatch(/sticky/);
    expect(phase.className).toMatch(/top-0/);
    expect(phase.className).toMatch(/\bh-8\b/);
    expect(phase.className).toMatch(/\bbg-muted\b/);
    expect(phase.className).not.toMatch(/\bbg-background\b/);

    const node = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(node.className).toMatch(/sticky/);
    expect(node.className).toMatch(/top-8/);
    expect(node.className).toMatch(/\bbg-muted\b/);
    expect(node.className).not.toMatch(/\bbg-background\b/);

    const step = screen.getByTestId("split-run-stream-line-step-clone");
    expect(step.className).toMatch(/sticky/);
    expect(step.className).toMatch(/top-\[3\.375rem\]/);
    expect(step.className).toMatch(/\bbg-muted\b/);
    expect(step.className).not.toMatch(/\bbg-background\b/);
  });

  it("shows agent text and a collapsed tool summary", async () => {
    const user = userEvent.setup();
    render(<PhaseLogCard phase={PHASE} expanded stream={PLANNING_STREAM} />);

    const note = screen.getByText(LONG_NOTE);
    expect(note).toBeInTheDocument();
    expect(note).not.toHaveClass("truncate");
    expect(note).toHaveClass("whitespace-normal");
    expect(note.parentElement?.className).toMatch(/\bbg-muted\b/);
    expect(note.parentElement?.className).not.toMatch(/\bbg-background\b/);
    const toolSummary = screen.getByRole("button", { name: "Ran 1 command" });
    expect(toolSummary).toBeInTheDocument();
    expect(toolSummary.className).toMatch(/\bbg-muted\b/);
    expect(toolSummary.className).not.toMatch(/\bbg-background\b/);
    expect(toolSummary.querySelector('[style*="ch"]')).toBeNull();
    expect(screen.getByRole("button", { name: "Read 1 file" })).toBeInTheDocument();
    expect(screen.queryByText("cat /tmp/ORDER.md")).not.toBeInTheDocument();
    expect(screen.queryByText("LineListCard.tsx")).not.toBeInTheDocument();
    expect(screen.queryByText("note")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ran 1 command" }));

    const stream = screen.getByTestId("split-run-stream-plan");
    expect(within(stream).getByText("cat /tmp/ORDER.md")).toBeInTheDocument();
    expect(within(stream).getByText("cat /tmp/ORDER.md").closest("ol")).not.toHaveClass("pl-2");
    expect(within(stream).queryByText(/## Goal/)).not.toBeInTheDocument();
    expect(within(stream).queryByText("LineListCard.tsx")).not.toBeInTheDocument();

    await user.click(within(stream).getByText("cat /tmp/ORDER.md"));
    expect(within(stream).getByText(/## Goal/)).toBeInTheDocument();
    expect(within(stream).getByText(/Add a menu/)).toBeInTheDocument();
    expect(screen.queryByTestId("split-run-stream-output-indent")).not.toBeInTheDocument();

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

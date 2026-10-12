import { describe, expect, it } from "bun:test";
import { render, screen } from "@testing-library/react";

import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import type { AgentActivity } from "../agentActivity";
import { notesFromLiveLogSections } from "../streamNotesFromLiveLog";
import { AgentStepMarkers } from "./AgentStepList";
import { agentStepsFromNotes, type AgentStep, type AutomationStage } from "./automationsViewModel";

const LONG_PROMPT =
  "You refine draft tasks so a coding agent can build them in one run. You read the task and the repository.";

function stageWithStep(step: AgentStep): AutomationStage {
  return {
    id: "implement",
    name: "Implementation",
    componentName: "Implementation",
    status: "passed",
    statusLabel: "Passed",
    duration: "1m",
    checks: [],
    outputs: { pullRequests: [], artifacts: [] },
    plumbing: [],
    agentSteps: [step],
    steps: [step],
    rawLog: "",
  };
}

function stageWithSteps(steps: AgentStep[], status: AutomationStage["status"] = "passed"): AutomationStage {
  return {
    ...stageWithStep(steps[0] ?? stepPlaceholder()),
    status,
    statusLabel: status === "failed" ? "Failed" : "Passed",
    agentSteps: steps,
    steps,
  };
}

function stepPlaceholder(): AgentStep {
  return {
    id: "empty",
    title: "Empty",
    type: "prompt",
    status: "passed",
    summary: "",
    toolCount: 0,
    events: [],
  };
}

function section(overrides: Partial<CommandSection> & Pick<CommandSection, "index" | "text" | "kind">): CommandSection {
  return {
    preview: "",
    lines: [],
    events: [],
    status: "passed",
    duration_ms: 20,
    started_at: 1,
    collapsed: true,
    ...overrides,
  };
}

function stepsFromSections(sections: CommandSection[], runStatus: "passed" | "failed"): AgentStep[] {
  return agentStepsFromNotes(notesFromLiveLogSections("agent", sections, runStatus), runStatus);
}

describe("AgentStepMarkers", () => {
  it("keeps the step duration on one line and shows it on hover", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "passed",
          duration: "23m 34s",
          summary: "",
          toolCount: 0,
          events: [],
        })}
      />,
    );

    const duration = screen.getByText("23m 34s");
    expect(duration).toHaveClass("whitespace-nowrap");
    expect(duration).toHaveClass("opacity-0");
    expect(duration).toHaveClass("group-hover/step:opacity-100");
    expect(duration).not.toHaveClass("w-12");
  });

  it("keeps a long prompt subtitle on one truncated line", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "refine",
          title: "Refine task",
          type: "prompt",
          status: "passed",
          summary: "",
          toolCount: 0,
          output: LONG_PROMPT,
          events: [],
        })}
      />,
    );

    const title = screen.getByText("Refine task");
    const subtitle = screen.getByText(LONG_PROMPT);
    expect(title).toHaveClass("truncate");
    expect(subtitle).toHaveClass("truncate");
    expect(subtitle).not.toHaveClass("break-words");
  });

  it("keeps Planning next step under a long live transcript", () => {
    const liveActivity: AgentActivity = {
      id: "live-1",
      provider: "runner",
      status: "running",
      sequence: 4,
      truncated: false,
      items: [
        {
          type: "content",
          id: "note-1",
          kind: "assistant",
          text: "Bun is not installed on the host.",
          status: "passed",
          truncated: false,
        },
        {
          type: "tool",
          id: "tool-1",
          kind: "read",
          name: "read",
          input: "README.md",
          output: "",
          outputStreams: [],
          status: "passed",
          truncated: false,
        },
      ],
    };

    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "running",
          summary: "",
          toolCount: 1,
          events: [],
        })}
        liveActive
        liveActivity={liveActivity}
      />,
    );

    const status = screen.getByRole("status", { name: "Planning next step…" });
    const log = screen.getByTestId("redesign-step-log-implementation");
    expect(status).toBeInTheDocument();
    expect(log).not.toContainElement(status);
  });

  it("shows Thinking when the console hides an empty thought row", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "running",
          summary: "",
          toolCount: 0,
          events: [],
        })}
        liveActive
        liveActivity={{
          id: "live-1",
          provider: "runner",
          status: "running",
          sequence: 2,
          truncated: false,
          items: [
            {
              type: "content",
              id: "thought-1",
              kind: "reasoning",
              text: "",
              status: "running",
              truncated: false,
            },
          ],
        }}
      />,
    );

    expect(screen.getByRole("status", { name: "Thinking" })).toBeInTheDocument();
    expect(screen.getByTestId("split-run-intent-thinking")).toHaveAccessibleName("Thinking");
  });

  it("names a streaming assistant reply Writing response", () => {
    render(
      <AgentStepMarkers
        stage={stageWithStep({
          id: "implementation",
          title: "Implementation",
          type: "prompt",
          status: "running",
          summary: "",
          toolCount: 0,
          events: [],
        })}
        liveActive
        liveActivity={{
          id: "live-1",
          provider: "runner",
          status: "running",
          sequence: 2,
          truncated: false,
          items: [
            {
              type: "content",
              id: "note-1",
              kind: "assistant",
              text: "Bun is not installed on the host.",
              status: "running",
              truncated: false,
            },
          ],
        }}
      />,
    );

    expect(screen.getByRole("status", { name: "Writing response…" })).toBeInTheDocument();
  });

  it("keeps a failed prompt red and does not mark the step Failed when the run passed", () => {
    render(
      <AgentStepMarkers
        expandSteps
        stage={stageWithSteps(
          stepsFromSections(
            [
              section({
                index: 1,
                text: "Refine Task",
                kind: "prompt",
                status: "failed",
                preview: "You refine draft tasks so a coding agent can build them in one run.",
                events: [{ kind: "note", text: "OpenCode started" }],
              }),
            ],
            "passed",
          ),
        )}
      />,
    );

    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
    expect(screen.getByTestId("agent-tool-agent-step-1-output")).toHaveAttribute("data-status", "failed");
  });

  it("shows Failed on the prompt step that stopped the run", () => {
    render(
      <AgentStepMarkers
        expandSteps
        stage={stageWithSteps(
          stepsFromSections(
            [
              section({
                index: 1,
                text: "Refine Task",
                kind: "prompt",
                status: "failed",
                preview: "You refine draft tasks so a coding agent can build them in one run.",
              }),
            ],
            "failed",
          ),
          "failed",
        )}
      />,
    );

    expect(screen.getByText("Failed")).toBeInTheDocument();
    expect(screen.getByTestId("agent-tool-agent-step-1-output")).toHaveAttribute("data-status", "failed");
  });

  it("does not show Failed on an earlier prompt when a later step stopped the run", () => {
    render(
      <AgentStepMarkers
        expandSteps
        stage={stageWithSteps(
          stepsFromSections(
            [
              section({
                index: 1,
                text: "Refine Task",
                kind: "prompt",
                status: "failed",
                preview: "You refine draft tasks so a coding agent can build them in one run.",
              }),
              section({
                index: 2,
                text: "Clone repository",
                kind: "bash",
                status: "failed",
                preview: "git clone https://example.com/repo.git",
              }),
            ],
            "failed",
          ),
          "failed",
        )}
      />,
    );

    expect(screen.getByText("Refine Task").parentElement).not.toHaveTextContent("Failed");
    expect(screen.getByText("Clone repository").parentElement).toHaveTextContent("Failed");
    expect(screen.getByTestId("agent-tool-agent-step-1-output")).toHaveAttribute("data-status", "failed");
  });

  it("shows Failed on a failed bash step", () => {
    render(
      <AgentStepMarkers
        stage={stageWithSteps(
          stepsFromSections(
            [
              section({
                index: 1,
                text: "Clone repository",
                kind: "bash",
                status: "failed",
                preview: "git clone https://example.com/repo.git",
              }),
            ],
            "passed",
          ),
        )}
      />,
    );

    expect(screen.getByText("Failed")).toBeInTheDocument();
  });
});

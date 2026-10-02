import { describe, expect, it } from "bun:test";
import { render, screen } from "@testing-library/react";

import type { AgentActivity } from "../agentActivity";
import { AgentStepMarkers } from "./AgentStepList";
import type { AgentStep, AutomationStage } from "./automationsViewModel";

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
});

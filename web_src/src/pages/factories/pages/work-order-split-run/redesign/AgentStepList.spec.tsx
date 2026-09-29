import { describe, expect, it } from "bun:test";
import { render, screen } from "@testing-library/react";

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
});

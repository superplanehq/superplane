import { describe, expect, it } from "bun:test";
import type { CanvasesCanvas } from "@/api-client";

import type { PlanningReviewDraft, PlanningReviewStep } from "../pages/planningReviewMockup";
import { applyDefaultRefinementPrompt, defaultRefineTaskStep } from "./refinementPrompt";

const DEFAULT_PROMPT = "Plan the task.\n\nTask:\n{{ root().data.workOrder }}";

const defaultStep: PlanningReviewStep = {
  name: "Refine Task",
  type: "prompt",
  prompt: DEFAULT_PROMPT,
  workingDirectory: "repo",
};

function draftWith(steps: PlanningReviewStep[]): PlanningReviewDraft {
  return {
    title: "Refine Task",
    components: [
      {
        id: "refine-task",
        title: "Refine Task",
        description: "",
        expanded: true,
        configuration: {
          model: "sonnet",
          credentials: { source: "integration", integration: { name: "claude" } },
          steps,
        },
        concurrency: { max: "1", key: "" },
      },
    ],
  };
}

function stepsOf(draft: PlanningReviewDraft): PlanningReviewStep[] {
  return draft.components[0].configuration.steps as PlanningReviewStep[];
}

describe("defaultRefineTaskStep", () => {
  it("reads only the Refine Task prompt step from the backlog canvas", () => {
    const spec = {
      nodes: [
        {
          id: "refine-task",
          configuration: {
            model: "opus",
            steps: [
              { name: "Clone repository", type: "bash", command: "git clone" },
              { ...defaultStep, command: "ignored" },
            ],
          },
        },
      ],
    } as CanvasesCanvas["spec"];

    expect(defaultRefineTaskStep(spec)).toEqual(defaultStep);
  });
});

describe("applyDefaultRefinementPrompt", () => {
  it("replaces a custom Refine Task prompt and keeps the rest of the agent", () => {
    const draft = draftWith([
      { name: "Clone repository", type: "bash", command: "git clone" },
      { name: "Refine Task", type: "prompt", prompt: "Custom prompt", workingDirectory: "repo" },
    ]);

    const next = applyDefaultRefinementPrompt(draft, defaultStep);

    expect(stepsOf(next)[0]).toEqual(stepsOf(draft)[0]);
    expect(stepsOf(next)[1]).toEqual({
      name: "Refine Task",
      type: "prompt",
      prompt: DEFAULT_PROMPT,
      workingDirectory: "repo",
    });
    expect(next.components[0].configuration.model).toBe("sonnet");
    expect(next.components[0].configuration.credentials).toEqual(draft.components[0].configuration.credentials);
  });

  it("replaces the prompt on a single renamed prompt step", () => {
    const draft = draftWith([
      { name: "Clone repository", type: "bash", command: "git clone" },
      { name: "Plan the work", type: "prompt", prompt: "Custom prompt", workingDirectory: "src" },
    ]);

    const next = applyDefaultRefinementPrompt(draft, defaultStep);

    expect(stepsOf(next)).toEqual([
      { name: "Clone repository", type: "bash", command: "git clone" },
      { name: "Plan the work", type: "prompt", prompt: DEFAULT_PROMPT, workingDirectory: "src" },
    ]);
  });

  it("appends the default step when several prompt steps are unrelated", () => {
    const draft = draftWith([
      { name: "Summarize", type: "prompt", prompt: "Summarize the task." },
      { name: "Score", type: "prompt", prompt: "Score the task." },
    ]);

    const next = applyDefaultRefinementPrompt(draft, { ...defaultStep, command: "do not copy" });

    expect(stepsOf(next)).toEqual([
      { name: "Summarize", type: "prompt", prompt: "Summarize the task." },
      { name: "Score", type: "prompt", prompt: "Score the task." },
      defaultStep,
    ]);
  });

  it("leaves the draft unchanged when the prompt already matches", () => {
    const draft = draftWith([
      { name: "Refine Task", type: "prompt", prompt: DEFAULT_PROMPT, workingDirectory: "elsewhere" },
    ]);

    expect(applyDefaultRefinementPrompt(draft, defaultStep)).toBe(draft);
  });
});

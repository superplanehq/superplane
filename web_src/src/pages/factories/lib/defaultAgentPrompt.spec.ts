import { describe, expect, it } from "bun:test";
import type { CanvasesCanvas } from "@/api-client";

import type { PlanningReviewDraft, PlanningReviewStep } from "../pages/planningReviewMockup";
import { applyDefaultAgentPrompts, defaultAgentSteps } from "./defaultAgentPrompt";

const REFINE_PROMPT = "Plan the task.\n\nTask:\n{{ root().data.workOrder }}";
const IMPLEMENT_PROMPT = "Implement the task.";
const PR_PROMPT = "Write the pull request title and description.";

const refineDefault: PlanningReviewStep = {
  name: "Refine Task",
  type: "prompt",
  prompt: REFINE_PROMPT,
  workingDirectory: "repo",
};

function draftWith(id: string, steps: PlanningReviewStep[]): PlanningReviewDraft {
  return {
    title: id,
    components: [
      {
        id,
        title: id,
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

describe("defaultAgentSteps", () => {
  it("reads ordered steps from the matching agent node", () => {
    const spec = {
      nodes: [
        {
          id: "other-agent",
          configuration: {
            steps: [{ name: "Other", type: "prompt", prompt: "Ignore this prompt." }],
          },
        },
        {
          id: "implementation-agent-no-issue",
          configuration: {
            model: "opus",
            steps: [
              { name: "Clone Repo", type: "bash", command: "git clone" },
              { name: "Implementation", type: "prompt", prompt: IMPLEMENT_PROMPT, workingDirectory: "repo" },
              { name: "Generate PR title and description", type: "prompt", prompt: PR_PROMPT },
            ],
          },
        },
      ],
    } as CanvasesCanvas["spec"];

    expect(defaultAgentSteps(spec, "implementation-agent-no-issue")).toEqual([
      { name: "Clone Repo", type: "bash", command: "git clone" },
      { name: "Implementation", type: "prompt", prompt: IMPLEMENT_PROMPT, workingDirectory: "repo" },
      { name: "Generate PR title and description", type: "prompt", prompt: PR_PROMPT },
    ]);
  });

  it("returns null when the agent node is missing", () => {
    const spec = {
      nodes: [{ id: "refine-task", configuration: { steps: [refineDefault] } }],
    } as CanvasesCanvas["spec"];

    expect(defaultAgentSteps(spec, "implementation-agent-no-issue")).toBeNull();
  });
});

describe("applyDefaultAgentPrompts", () => {
  it("replaces named prompt text and keeps the rest of the agent", () => {
    const draft = draftWith("refine-task", [
      { name: "Clone repository", type: "bash", command: "git clone" },
      { name: "Refine Task", type: "prompt", prompt: "Custom prompt", workingDirectory: "repo" },
    ]);

    const next = applyDefaultAgentPrompts(draft, [refineDefault]);

    expect(stepsOf(next)[0]).toEqual(stepsOf(draft)[0]);
    expect(stepsOf(next)[1]).toEqual({
      name: "Refine Task",
      type: "prompt",
      prompt: REFINE_PROMPT,
      workingDirectory: "repo",
    });
    expect(next.components[0].configuration.model).toBe("sonnet");
    expect(next.components[0].configuration.credentials).toEqual(draft.components[0].configuration.credentials);
  });

  it("replaces the prompt on a single renamed prompt step", () => {
    const draft = draftWith("refine-task", [
      { name: "Clone repository", type: "bash", command: "git clone" },
      { name: "Plan the work", type: "prompt", prompt: "Custom prompt", workingDirectory: "src" },
    ]);

    expect(stepsOf(applyDefaultAgentPrompts(draft, [refineDefault]))).toEqual([
      { name: "Clone repository", type: "bash", command: "git clone" },
      { name: "Plan the work", type: "prompt", prompt: REFINE_PROMPT, workingDirectory: "src" },
    ]);
  });

  it("replaces every named prompt on an implementation agent", () => {
    const draft = draftWith("implementation-agent-no-issue", [
      { name: "Clone Repo", type: "bash", command: "git clone" },
      { name: "Implementation", type: "prompt", prompt: "Custom implement", workingDirectory: "repo" },
      { name: "Generate PR title and description", type: "prompt", prompt: "Custom pr", workingDirectory: "repo" },
    ]);

    const next = applyDefaultAgentPrompts(draft, [
      { name: "Implementation", type: "prompt", prompt: IMPLEMENT_PROMPT },
      { name: "Generate PR title and description", type: "prompt", prompt: PR_PROMPT },
    ]);

    expect(stepsOf(next)).toEqual([
      { name: "Clone Repo", type: "bash", command: "git clone" },
      { name: "Implementation", type: "prompt", prompt: IMPLEMENT_PROMPT, workingDirectory: "repo" },
      { name: "Generate PR title and description", type: "prompt", prompt: PR_PROMPT, workingDirectory: "repo" },
    ]);
  });

  it("pairs renamed prompt steps in order when the counts match", () => {
    const draft = draftWith("implementation-agent-no-issue", [
      { name: "Do the work", type: "prompt", prompt: "Custom implement" },
      { name: "Describe the change", type: "prompt", prompt: "Custom pr" },
    ]);

    expect(
      stepsOf(
        applyDefaultAgentPrompts(draft, [
          { name: "Implementation", type: "prompt", prompt: IMPLEMENT_PROMPT },
          { name: "Generate PR title and description", type: "prompt", prompt: PR_PROMPT },
        ]),
      ),
    ).toEqual([
      { name: "Do the work", type: "prompt", prompt: IMPLEMENT_PROMPT },
      { name: "Describe the change", type: "prompt", prompt: PR_PROMPT },
    ]);
  });

  it("does not append a step when prompt counts differ", () => {
    const draft = draftWith("refine-task", [
      { name: "Summarize", type: "prompt", prompt: "Summarize the task." },
      { name: "Score", type: "prompt", prompt: "Score the task." },
    ]);

    expect(applyDefaultAgentPrompts(draft, [refineDefault])).toBe(draft);
  });

  it("does not assign the first default when one of two prompts remains and is renamed", () => {
    const draft = draftWith("implementation-agent-no-issue", [
      { name: "Describe the change", type: "prompt", prompt: "Custom pr" },
    ]);

    expect(
      applyDefaultAgentPrompts(draft, [
        { name: "Implementation", type: "prompt", prompt: IMPLEMENT_PROMPT },
        { name: "Generate PR title and description", type: "prompt", prompt: PR_PROMPT },
      ]),
    ).toBe(draft);
  });

  it("restores deleted prompt steps when none remain", () => {
    const draft = draftWith("refine-task", [{ name: "Clone repository", type: "bash", command: "git clone" }]);

    expect(stepsOf(applyDefaultAgentPrompts(draft, [refineDefault]))).toEqual([
      { name: "Clone repository", type: "bash", command: "git clone" },
      refineDefault,
    ]);
  });

  it("puts restored prompts back before the bash steps that follow them", () => {
    const draft = draftWith("implementation-agent-no-issue", [
      { name: "Clone Repo", type: "bash", command: "git clone" },
      { name: "Commit and Push", type: "bash", command: "git push" },
      { name: "Push output", type: "bash", command: "emit output" },
    ]);
    const implementation: PlanningReviewStep = {
      name: "Implementation",
      type: "prompt",
      prompt: IMPLEMENT_PROMPT,
      workingDirectory: "repo",
    };
    const pullRequest: PlanningReviewStep = {
      name: "Generate PR title and description",
      type: "prompt",
      prompt: PR_PROMPT,
    };

    expect(
      stepsOf(
        applyDefaultAgentPrompts(draft, [
          { name: "Clone Repo", type: "bash", command: "git clone --depth 1" },
          implementation,
          { name: "Commit and Push", type: "bash", command: "git push -u origin HEAD" },
          pullRequest,
          { name: "Push output", type: "bash", command: "jq ." },
        ]),
      ),
    ).toEqual([
      { name: "Clone Repo", type: "bash", command: "git clone" },
      implementation,
      { name: "Commit and Push", type: "bash", command: "git push" },
      pullRequest,
      { name: "Push output", type: "bash", command: "emit output" },
    ]);
  });

  it("puts restored prompts before a remaining later bash step", () => {
    const draft = draftWith("implementation-agent-no-issue", [
      { name: "Push output", type: "bash", command: "emit output" },
    ]);
    const implementation: PlanningReviewStep = {
      name: "Implementation",
      type: "prompt",
      prompt: IMPLEMENT_PROMPT,
      workingDirectory: "repo",
    };
    const pullRequest: PlanningReviewStep = {
      name: "Generate PR title and description",
      type: "prompt",
      prompt: PR_PROMPT,
    };

    expect(
      stepsOf(
        applyDefaultAgentPrompts(draft, [
          { name: "Clone Repo", type: "bash", command: "git clone --depth 1" },
          implementation,
          { name: "Commit and Push", type: "bash", command: "git push -u origin HEAD" },
          pullRequest,
          { name: "Push output", type: "bash", command: "jq ." },
        ]),
      ),
    ).toEqual([implementation, pullRequest, { name: "Push output", type: "bash", command: "emit output" }]);
  });

  it("restores a prompt before the later step when an earlier step has the same name", () => {
    const draft = draftWith("implementation-agent-no-issue", [
      { name: "Format JS and Go code", type: "bash", command: "format early" },
      { name: "Checkout branch", type: "bash", command: "git clone" },
      { name: "Format JS and Go code", type: "bash", command: "format late" },
    ]);
    const implementation: PlanningReviewStep = {
      name: "Implementation",
      type: "prompt",
      prompt: IMPLEMENT_PROMPT,
      workingDirectory: "repo",
    };

    expect(
      stepsOf(
        applyDefaultAgentPrompts(draft, [
          { name: "Format JS and Go code", type: "bash", command: "format first" },
          { name: "Checkout branch", type: "bash", command: "git clone --depth 1" },
          implementation,
          { name: "Format JS and Go code", type: "bash", command: "format second" },
        ]),
      ),
    ).toEqual([
      { name: "Format JS and Go code", type: "bash", command: "format early" },
      { name: "Checkout branch", type: "bash", command: "git clone" },
      implementation,
      { name: "Format JS and Go code", type: "bash", command: "format late" },
    ]);
  });

  it("replaces the prompt step when a bash step has the same name", () => {
    const draft = draftWith("refine-task", [
      { name: "Refine Task", type: "bash", command: "echo keep" },
      { name: "Plan the work", type: "prompt", prompt: "Custom prompt", workingDirectory: "src" },
    ]);

    expect(stepsOf(applyDefaultAgentPrompts(draft, [refineDefault]))).toEqual([
      { name: "Refine Task", type: "bash", command: "echo keep" },
      { name: "Plan the work", type: "prompt", prompt: REFINE_PROMPT, workingDirectory: "src" },
    ]);
  });

  it("leaves the draft unchanged when the prompt already matches", () => {
    const draft = draftWith("refine-task", [
      { name: "Refine Task", type: "prompt", prompt: REFINE_PROMPT, workingDirectory: "elsewhere" },
    ]);

    expect(applyDefaultAgentPrompts(draft, [refineDefault])).toBe(draft);
  });
});

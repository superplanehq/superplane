import { describe, expect, it, vi } from "vitest";

import {
  runWorkspaceNextStepAction,
  workspaceNextSteps,
  workspaceNextStepsProgressCopy,
} from "./workspaceNextStepCatalog";

const ready = {
  onboardingComplete: true,
  canConfigure: true,
  takenPRFeedbackSources: [] as const,
};

describe("workspaceNextSteps", () => {
  it("lists comments and status-check tasks as not done when neither handler exists", () => {
    expect(workspaceNextSteps(ready)).toEqual([
      expect.objectContaining({ id: "pr-comments-handler", done: false }),
      expect.objectContaining({ id: "pr-checks-handler", done: false }),
    ]);
  });

  it("keeps a configured task in the list as done", () => {
    expect(
      workspaceNextSteps({
        ...ready,
        takenPRFeedbackSources: ["discussion"],
      }),
    ).toEqual([
      expect.objectContaining({ id: "pr-comments-handler", done: true }),
      expect.objectContaining({ id: "pr-checks-handler", done: false }),
    ]);
  });

  it("hides the list when every handler is configured", () => {
    expect(
      workspaceNextSteps({
        ...ready,
        takenPRFeedbackSources: ["discussion", "checks"],
      }),
    ).toEqual([]);
  });

  it("hides the list before onboarding is complete or when the user cannot configure", () => {
    expect(workspaceNextSteps({ ...ready, onboardingComplete: false })).toEqual([]);
    expect(workspaceNextSteps({ ...ready, canConfigure: false })).toEqual([]);
  });
});

describe("workspaceNextStepsProgressCopy", () => {
  it("names how many tasks are complete", () => {
    expect(workspaceNextStepsProgressCopy(1, 2)).toBe("1 of 2 complete");
  });
});

describe("runWorkspaceNextStepAction", () => {
  it("starts the matching PR feedback source", () => {
    const configurePRFeedback = vi.fn();
    runWorkspaceNextStepAction({ type: "configure-pr-feedback", sourceId: "checks" }, { configurePRFeedback });
    expect(configurePRFeedback).toHaveBeenCalledWith("checks");
  });
});

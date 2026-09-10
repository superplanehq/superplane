import { describe, expect, it, vi } from "vitest";

import {
  runWorkspaceNextStepAction,
  workspaceNextStepBanner,
  workspaceNextSteps,
  workspaceNextStepsProgressCopy,
} from "./workspaceNextStepCatalog";

const ready = {
  onboardingComplete: true,
  canConfigure: true,
  takenPRFeedbackSources: [] as const,
};

describe("workspaceNextSteps", () => {
  it("lists the comments task as not done when no handler exists", () => {
    expect(workspaceNextSteps(ready)).toEqual([expect.objectContaining({ id: "pr-comments-handler", done: false })]);
  });

  it("hides the list when the comments handler is configured", () => {
    expect(
      workspaceNextSteps({
        ...ready,
        takenPRFeedbackSources: ["discussion"],
      }),
    ).toEqual([]);
  });

  it("hides the list before onboarding is complete or when the user cannot configure", () => {
    expect(workspaceNextSteps({ ...ready, onboardingComplete: false })).toEqual([]);
    expect(workspaceNextSteps({ ...ready, canConfigure: false })).toEqual([]);
  });
});

describe("workspaceNextStepBanner", () => {
  it("personalizes the banner for comments when nothing is configured", () => {
    const banner = workspaceNextStepBanner(workspaceNextSteps(ready));
    expect(banner?.activeStep.id).toBe("pr-comments-handler");
    expect(banner?.doneCount).toBe(2);
    expect(banner?.totalCount).toBe(3);
    expect(banner?.title).toBe("How should pull request comments be handled?");
    expect(banner?.description).toBe(
      "SuperPlane can implement tasks and open pull requests, but pull request reviews are not handled yet.",
    );
    expect(banner?.ctaLabel).toBe("Configure");
  });

  it("hides the banner after comments are configured", () => {
    expect(
      workspaceNextStepBanner(
        workspaceNextSteps({
          ...ready,
          takenPRFeedbackSources: ["discussion"],
        }),
      ),
    ).toBeNull();
  });
});

describe("workspaceNextStepsProgressCopy", () => {
  it("names how many tasks are complete", () => {
    expect(workspaceNextStepsProgressCopy(2, 3)).toBe("2/3");
  });
});

describe("runWorkspaceNextStepAction", () => {
  it("opens the matching PR feedback setup page", () => {
    const openPRFeedbackSetup = vi.fn();
    runWorkspaceNextStepAction({ type: "open-pr-feedback-setup", sourceId: "discussion" }, { openPRFeedbackSetup });
    expect(openPRFeedbackSetup).toHaveBeenCalledWith("discussion");
  });
});

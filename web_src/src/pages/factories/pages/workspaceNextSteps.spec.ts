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
  it("lists comments and status-check tasks as not done when neither handler exists", () => {
    expect(workspaceNextSteps(ready)).toEqual([
      expect.objectContaining({ id: "pr-comments-handler", done: false }),
      expect.objectContaining({ id: "pr-checks-handler", done: false }),
    ]);
  });

  it("marks the provisioned comments handler as done and keeps status checks open", () => {
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

describe("workspaceNextStepBanner", () => {
  it("personalizes the banner for comments when nothing is configured", () => {
    const banner = workspaceNextStepBanner(workspaceNextSteps(ready));
    expect(banner?.activeStep.id).toBe("pr-comments-handler");
    expect(banner?.doneCount).toBe(0);
    expect(banner?.title).toBe("How should pull request comments be handled?");
    expect(banner?.worksCopy).toBe(
      "This SuperPlane workspace can implement tasks and open pull requests for them, but it will not start fixes from pull request reviews yet.",
    );
    expect(banner?.missingCopy).toBe("Configure how pull request reviews should be handled next.");
    expect(banner?.ctaLabel).toBe("Configure comments");
  });

  it("personalizes the banner for status checks after comments are done", () => {
    const banner = workspaceNextStepBanner(
      workspaceNextSteps({
        ...ready,
        takenPRFeedbackSources: ["discussion"],
      }),
    );
    expect(banner?.activeStep.id).toBe("pr-checks-handler");
    expect(banner?.doneCount).toBe(1);
    expect(banner?.title).toBe("How should failing status checks be handled?");
    expect(banner?.worksCopy).toBe("Pull request comments and reviews can start a fix.");
    expect(banner?.missingCopy).toContain("will not wait for failing status checks");
    expect(banner?.ctaLabel).toBe("Configure status checks");
  });
});

describe("workspaceNextStepsProgressCopy", () => {
  it("names how many tasks are complete", () => {
    expect(workspaceNextStepsProgressCopy(1, 2)).toBe("1/2");
  });
});

describe("runWorkspaceNextStepAction", () => {
  it("opens the matching PR feedback setup page", () => {
    const openPRFeedbackSetup = vi.fn();
    runWorkspaceNextStepAction({ type: "open-pr-feedback-setup", sourceId: "checks" }, { openPRFeedbackSetup });
    expect(openPRFeedbackSetup).toHaveBeenCalledWith("checks");
  });
});

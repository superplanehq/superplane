import { describe, expect, it, vi } from "bun:test";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import {
  isWorkspaceNextStepDeferred,
  isWorkspaceNextStepsQueryReady,
  runWorkspaceNextStepAction,
  shouldForgetDeferredWorkspaceNextStep,
  workspaceNextStepBanner,
  workspaceNextSteps,
  workspaceNextStepsProgressCopy,
} from "./workspaceNextStepCatalog";

const ready = {
  onboardingComplete: true,
  canConfigure: true,
  prFeedbackHandlersReady: true,
  planningSetupReady: true,
  planningSetupCompleted: false,
  takenPRFeedbackSources: [] as const,
};

const afterPlanning = {
  ...ready,
  planningSetupCompleted: true,
};

describe("workspaceNextSteps", () => {
  it("lists Planning first, then comments and status-check tasks", () => {
    expect(workspaceNextSteps(ready)).toEqual([
      expect.objectContaining({ id: "planning-setup", done: false, canDefer: false }),
      expect.objectContaining({ id: "pr-comments-handler", done: false }),
      expect.objectContaining({ id: "pr-checks-handler", done: false }),
    ]);
  });

  it("marks Planning done only after setupCompleted", () => {
    expect(
      workspaceNextSteps({
        ...ready,
        planningSetupCompleted: true,
      }),
    ).toEqual([
      expect.objectContaining({ id: "planning-setup", done: true }),
      expect.objectContaining({ id: "pr-comments-handler", done: false }),
      expect.objectContaining({ id: "pr-checks-handler", done: false }),
    ]);
  });

  it("marks comments done and keeps status checks open", () => {
    expect(
      workspaceNextSteps({
        ...afterPlanning,
        takenPRFeedbackSources: ["discussion"],
      }),
    ).toEqual([
      expect.objectContaining({ id: "planning-setup", done: true }),
      expect.objectContaining({ id: "pr-comments-handler", done: true }),
      expect.objectContaining({ id: "pr-checks-handler", done: false }),
    ]);
  });

  it("hides the list when every next step is done", () => {
    expect(
      workspaceNextSteps({
        ...afterPlanning,
        takenPRFeedbackSources: ["discussion", "checks"],
      }),
    ).toEqual([]);
  });

  it("hides the list before onboarding is complete or when the user cannot configure", () => {
    expect(workspaceNextSteps({ ...ready, onboardingComplete: false })).toEqual([]);
    expect(workspaceNextSteps({ ...ready, canConfigure: false })).toEqual([]);
  });

  it("hides the list until handler status and factory Planning are ready", () => {
    expect(workspaceNextSteps({ ...ready, prFeedbackHandlersReady: false })).toEqual([]);
    expect(workspaceNextSteps({ ...ready, planningSetupReady: false })).toEqual([]);
    expect(workspaceNextSteps({ ...ready, prFeedbackHandlersReady: true, planningSetupReady: true })).toEqual([
      expect.objectContaining({ id: "planning-setup", done: false }),
      expect.objectContaining({ id: "pr-comments-handler", done: false }),
      expect.objectContaining({ id: "pr-checks-handler", done: false }),
    ]);
  });
});

describe("isWorkspaceNextStepsQueryReady", () => {
  it("stays hidden while the query is pending or failed without data", () => {
    expect(isWorkspaceNextStepsQueryReady({ isPending: true, isError: false })).toBe(false);
    expect(isWorkspaceNextStepsQueryReady({ isPending: false, isError: true })).toBe(false);
    expect(isWorkspaceNextStepsQueryReady({ isPending: false, isError: true, data: undefined })).toBe(false);
  });

  it("is ready after a successful load, including an empty list", () => {
    expect(isWorkspaceNextStepsQueryReady({ isPending: false, isError: false, data: [] })).toBe(true);
    expect(isWorkspaceNextStepsQueryReady({ isPending: false, data: [{ id: "handler-1" }] })).toBe(true);
  });

  it("uses cached handlers when a later fetch fails", () => {
    expect(isWorkspaceNextStepsQueryReady({ isPending: false, isError: true, data: [{ id: "handler-1" }] })).toBe(true);
  });
});

describe("workspaceNextStepBanner", () => {
  it("personalizes the banner for Planning when setup is not confirmed", () => {
    const banner = workspaceNextStepBanner(workspaceNextSteps(ready));
    expect(banner?.activeStep.id).toBe("planning-setup");
    expect(banner?.doneCount).toBe(2);
    expect(banner?.totalCount).toBe(5);
    expect(banner?.title).toBe(PLANNING_SETTINGS_COPY.wizardPageTitle);
    expect(banner?.badgeLabel).toBe(PLANNING_SETTINGS_COPY.wizardBadgeLabel);
    expect(banner?.description).toBe(PLANNING_SETTINGS_COPY.wizardBannerDescription);
    expect(banner?.ctaLabel).toBe("Configure");
    expect(banner?.canDefer).toBe(false);
  });

  it("personalizes the banner for comments after Planning is confirmed", () => {
    const banner = workspaceNextStepBanner(workspaceNextSteps(afterPlanning));
    expect(banner?.activeStep.id).toBe("pr-comments-handler");
    expect(banner?.doneCount).toBe(3);
    expect(banner?.totalCount).toBe(5);
    expect(banner?.title).toBe("How should pull request comments be handled?");
    expect(banner?.description).toBe(
      "SuperPlane can implement tasks and open pull requests, but pull request reviews are not handled yet.",
    );
    expect(banner?.ctaLabel).toBe("Configure");
    expect(banner?.canDefer).toBe(false);
  });

  it("personalizes the banner for status checks after comments are configured", () => {
    const banner = workspaceNextStepBanner(
      workspaceNextSteps({
        ...afterPlanning,
        takenPRFeedbackSources: ["discussion"],
      }),
    );
    expect(banner?.activeStep.id).toBe("pr-checks-handler");
    expect(banner?.doneCount).toBe(4);
    expect(banner?.totalCount).toBe(5);
    expect(banner?.title).toBe("How should failing status checks be handled?");
    expect(banner?.badgeLabel).toBe("Configure status checks");
    expect(banner?.description).toContain("automatically fix failing pull request status checks");
    expect(banner?.ctaLabel).toBe("Configure");
    expect(banner?.canDefer).toBe(true);
  });

  it("hides the banner after every next step is done", () => {
    expect(
      workspaceNextStepBanner(
        workspaceNextSteps({
          ...afterPlanning,
          takenPRFeedbackSources: ["discussion", "checks"],
        }),
      ),
    ).toBeNull();
  });
});

describe("workspaceNextStepsProgressCopy", () => {
  it("names how many tasks are complete", () => {
    expect(workspaceNextStepsProgressCopy(2, 5)).toBe("2/5");
  });
});

describe("isWorkspaceNextStepDeferred", () => {
  it("defers only the optional status-checks banner", () => {
    const planning = workspaceNextStepBanner(workspaceNextSteps(ready));
    const comments = workspaceNextStepBanner(workspaceNextSteps(afterPlanning));
    const checks = workspaceNextStepBanner(
      workspaceNextSteps({
        ...afterPlanning,
        takenPRFeedbackSources: ["discussion"],
      }),
    );

    expect(isWorkspaceNextStepDeferred(planning, "pr-checks-handler")).toBe(false);
    expect(isWorkspaceNextStepDeferred(comments, "pr-checks-handler")).toBe(false);
    expect(isWorkspaceNextStepDeferred(checks, "pr-checks-handler")).toBe(true);
    expect(isWorkspaceNextStepDeferred(checks, "pr-comments-handler")).toBe(false);
    expect(isWorkspaceNextStepDeferred(null, "pr-checks-handler")).toBe(false);
  });
});

describe("shouldForgetDeferredWorkspaceNextStep", () => {
  it("forgets a deferred checks step after that handler exists", () => {
    expect(shouldForgetDeferredWorkspaceNextStep("pr-checks-handler", ["discussion"])).toBe(false);
    expect(shouldForgetDeferredWorkspaceNextStep("pr-checks-handler", ["discussion", "checks"])).toBe(true);
    expect(shouldForgetDeferredWorkspaceNextStep(null, ["checks"])).toBe(false);
  });
});

describe("runWorkspaceNextStepAction", () => {
  it("opens the Planning setup page", () => {
    const openPlanningSetup = vi.fn();
    const openPRFeedbackSetup = vi.fn();
    runWorkspaceNextStepAction({ type: "open-planning-setup" }, { openPlanningSetup, openPRFeedbackSetup });
    expect(openPlanningSetup).toHaveBeenCalledOnce();
    expect(openPRFeedbackSetup).not.toHaveBeenCalled();
  });

  it("opens the matching PR feedback setup page", () => {
    const openPlanningSetup = vi.fn();
    const openPRFeedbackSetup = vi.fn();
    runWorkspaceNextStepAction(
      { type: "open-pr-feedback-setup", sourceId: "discussion" },
      { openPlanningSetup, openPRFeedbackSetup },
    );
    expect(openPRFeedbackSetup).toHaveBeenCalledWith("discussion");
    expect(openPlanningSetup).not.toHaveBeenCalled();
  });
});

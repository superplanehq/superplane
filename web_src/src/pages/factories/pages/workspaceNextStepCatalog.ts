import { type PRFeedbackSourceId } from "./prFeedbackSettingsModel";

export type WorkspaceNextStepId = "pr-comments-handler" | "pr-checks-handler";

export type WorkspaceNextStepAction = { type: "open-pr-feedback-setup"; sourceId: PRFeedbackSourceId };

export interface WorkspaceNextStep {
  id: WorkspaceNextStepId;
  /** Short name used in lists and tests. */
  title: string;
  /** Banner header when this step is the next action. */
  bannerTitle: string;
  /** What already works before this step is configured. */
  worksCopy: string;
  /** What stays missing until the user configures this step. */
  missingCopy: string;
  ctaLabel: string;
  action: WorkspaceNextStepAction;
  done: boolean;
}

export interface WorkspaceNextStepContext {
  onboardingComplete: boolean;
  canConfigure: boolean;
  takenPRFeedbackSources: readonly PRFeedbackSourceId[];
}

export interface WorkspaceNextStepBanner {
  doneCount: number;
  totalCount: number;
  /** First incomplete step; drives the single CTA. */
  activeStep: WorkspaceNextStep;
  title: string;
  worksCopy: string;
  missingCopy: string;
  ctaLabel: string;
}

const WORKSPACE_NEXT_STEPS: Array<
  Omit<WorkspaceNextStep, "done"> & { isDone: (ctx: WorkspaceNextStepContext) => boolean }
> = [
  {
    id: "pr-comments-handler",
    title: "Comments handler",
    bannerTitle: "How should pull request comments be handled?",
    worksCopy:
      "This SuperPlane workspace can implement tasks and open pull requests for them, but it will not start fixes from pull request reviews yet.",
    missingCopy: "Configure how pull request reviews should be handled next.",
    ctaLabel: "Configure comments",
    action: { type: "open-pr-feedback-setup", sourceId: "discussion" },
    isDone: (ctx) => ctx.takenPRFeedbackSources.includes("discussion"),
  },
  {
    id: "pr-checks-handler",
    title: "Status checks handler",
    bannerTitle: "How should failing status checks be handled?",
    worksCopy: "Pull request comments and reviews can start a fix.",
    missingCopy: "SuperPlane will not wait for failing status checks until you configure this.",
    ctaLabel: "Configure status checks",
    action: { type: "open-pr-feedback-setup", sourceId: "checks" },
    isDone: (ctx) => ctx.takenPRFeedbackSources.includes("checks"),
  },
];

export function workspaceNextStepsProgressCopy(done: number, total: number): string {
  return `${done}/${total}`;
}

export function workspaceNextSteps(ctx: WorkspaceNextStepContext): WorkspaceNextStep[] {
  if (!ctx.onboardingComplete || !ctx.canConfigure) {
    return [];
  }
  const steps = WORKSPACE_NEXT_STEPS.map(({ isDone, ...step }) => ({
    ...step,
    done: isDone(ctx),
  }));
  if (steps.every((step) => step.done)) {
    return [];
  }
  return steps;
}

export function workspaceNextStepBanner(steps: WorkspaceNextStep[]): WorkspaceNextStepBanner | null {
  const activeStep = steps.find((step) => !step.done);
  if (!activeStep) {
    return null;
  }
  return {
    doneCount: steps.filter((step) => step.done).length,
    totalCount: steps.length,
    activeStep,
    title: activeStep.bannerTitle,
    worksCopy: activeStep.worksCopy,
    missingCopy: activeStep.missingCopy,
    ctaLabel: activeStep.ctaLabel,
  };
}

export function runWorkspaceNextStepAction(
  action: WorkspaceNextStepAction,
  handlers: { openPRFeedbackSetup: (sourceId: PRFeedbackSourceId) => void },
) {
  if (action.type === "open-pr-feedback-setup") {
    handlers.openPRFeedbackSetup(action.sourceId);
  }
}

import { PR_FEEDBACK_SETTINGS_COPY, type PRFeedbackSourceId } from "./prFeedbackSettingsModel";

export type WorkspaceNextStepId = "pr-comments-handler" | "pr-checks-handler";

export type WorkspaceNextStepAction = { type: "open-pr-feedback-setup"; sourceId: PRFeedbackSourceId };

export interface WorkspaceNextStep {
  id: WorkspaceNextStepId;
  /** Short name used in lists and tests. */
  title: string;
  /** Banner header when this step is the next action. */
  bannerTitle: string;
  /** Short call to action for the minimized header badge. */
  badgeLabel: string;
  /** Banner body when this step is the next action. */
  description: string;
  ctaLabel: string;
  /** Optional steps can hide the banner without completing the task. */
  canDefer: boolean;
  action: WorkspaceNextStepAction;
  done: boolean;
}

export interface WorkspaceNextStepContext {
  onboardingComplete: boolean;
  canConfigure: boolean;
  takenPRFeedbackSources: readonly PRFeedbackSourceId[];
  /** Stay hidden until PR feedback handlers have loaded. Also hide when the query fails without cached data. */
  ready?: boolean;
}

export interface WorkspaceNextStepBanner {
  doneCount: number;
  totalCount: number;
  /** First incomplete step; drives the single CTA. */
  activeStep: WorkspaceNextStep;
  title: string;
  badgeLabel: string;
  description: string;
  ctaLabel: string;
  canDefer: boolean;
}

/** GitHub and the repository are already set during onboarding. */
const ONBOARDING_COMPLETED_STEP_COUNT = 2;

type WorkspaceNextStepDefinition = Omit<WorkspaceNextStep, "done"> & {
  isDone: (ctx: WorkspaceNextStepContext) => boolean;
};

const COMMENTS_NEXT_STEP: WorkspaceNextStepDefinition = {
  id: "pr-comments-handler",
  title: "Comments handler",
  bannerTitle: PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleComments,
  badgeLabel: "Configure pull request comments",
  description: "SuperPlane can implement tasks and open pull requests, but pull request reviews are not handled yet.",
  ctaLabel: "Configure",
  canDefer: false,
  action: { type: "open-pr-feedback-setup", sourceId: "discussion" },
  isDone: (ctx) => ctx.takenPRFeedbackSources.includes("discussion"),
};

const CHECKS_NEXT_STEP: WorkspaceNextStepDefinition = {
  id: "pr-checks-handler",
  title: "Status checks handler",
  bannerTitle: PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleChecks,
  badgeLabel: "Configure status checks",
  description: [
    "SuperPlane can implement tasks, open pull requests, and address pull request reviews.",
    "Do you also want SuperPlane to automatically fix failing pull request status checks?",
  ].join("\n"),
  ctaLabel: "Configure",
  canDefer: true,
  action: { type: "open-pr-feedback-setup", sourceId: "checks" },
  isDone: (ctx) => ctx.takenPRFeedbackSources.includes("checks"),
};

export const WORKSPACE_NEXT_STEPS_COPY = {
  later: "Later",
  restoreLabel: (title: string) => `Show next steps. ${title}`,
} as const;

const WORKSPACE_NEXT_STEPS = [COMMENTS_NEXT_STEP, CHECKS_NEXT_STEP];

export function workspaceNextStepsProgressCopy(done: number, total: number): string {
  return `${done}/${total}`;
}

export function isWorkspaceNextStepsQueryReady(query: {
  isPending: boolean;
  isError?: boolean;
  data?: unknown;
}): boolean {
  if (query.isPending) {
    return false;
  }
  return query.data !== undefined || query.isError !== true;
}

export function workspaceNextSteps(ctx: WorkspaceNextStepContext): WorkspaceNextStep[] {
  if (ctx.ready === false || !ctx.onboardingComplete || !ctx.canConfigure) {
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
    doneCount: ONBOARDING_COMPLETED_STEP_COUNT + steps.filter((step) => step.done).length,
    totalCount: ONBOARDING_COMPLETED_STEP_COUNT + steps.length,
    activeStep,
    title: activeStep.bannerTitle,
    badgeLabel: activeStep.badgeLabel,
    description: activeStep.description,
    ctaLabel: activeStep.ctaLabel,
    canDefer: activeStep.canDefer,
  };
}

export function isWorkspaceNextStepDeferred(
  banner: WorkspaceNextStepBanner | null,
  deferredStepId: WorkspaceNextStepId | null,
): boolean {
  return Boolean(banner?.canDefer && deferredStepId === banner.activeStep.id);
}

export function shouldForgetDeferredWorkspaceNextStep(
  deferredStepId: WorkspaceNextStepId | null,
  takenPRFeedbackSources: readonly PRFeedbackSourceId[],
): boolean {
  return deferredStepId === "pr-checks-handler" && takenPRFeedbackSources.includes("checks");
}

export function runWorkspaceNextStepAction(
  action: WorkspaceNextStepAction,
  handlers: { openPRFeedbackSetup: (sourceId: PRFeedbackSourceId) => void },
) {
  if (action.type === "open-pr-feedback-setup") {
    handlers.openPRFeedbackSetup(action.sourceId);
  }
}

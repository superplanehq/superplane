import {
  CHECKS_PR_FEEDBACK_NEXT_STEP_AVAILABLE,
  PR_FEEDBACK_SETTINGS_COPY,
  type PRFeedbackSourceId,
} from "./prFeedbackSettingsModel";

export type WorkspaceNextStepId = "pr-comments-handler" | "pr-checks-handler";

export type WorkspaceNextStepAction = { type: "open-pr-feedback-setup"; sourceId: PRFeedbackSourceId };

export interface WorkspaceNextStep {
  id: WorkspaceNextStepId;
  /** Short name used in lists and tests. */
  title: string;
  /** Banner header when this step is the next action. */
  bannerTitle: string;
  /** Banner body when this step is the next action. */
  description: string;
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
  description: string;
  ctaLabel: string;
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
  description: "SuperPlane can implement tasks and open pull requests, but pull request reviews are not handled yet.",
  ctaLabel: "Configure",
  action: { type: "open-pr-feedback-setup", sourceId: "discussion" },
  isDone: (ctx) => ctx.takenPRFeedbackSources.includes("discussion"),
};

const CHECKS_NEXT_STEP: WorkspaceNextStepDefinition = {
  id: "pr-checks-handler",
  title: "Status checks handler",
  bannerTitle: PR_FEEDBACK_SETTINGS_COPY.wizardPageTitleChecks,
  description: [
    "SuperPlane can implement tasks, open pull requests, and address pull request reviews.",
    "Do you also want SuperPlane to automatically fix failing pull request status checks?",
  ].join("\n"),
  ctaLabel: "Configure",
  action: { type: "open-pr-feedback-setup", sourceId: "checks" },
  isDone: (ctx) => ctx.takenPRFeedbackSources.includes("checks"),
};

const WORKSPACE_NEXT_STEPS = [
  COMMENTS_NEXT_STEP,
  ...(CHECKS_PR_FEEDBACK_NEXT_STEP_AVAILABLE ? [CHECKS_NEXT_STEP] : []),
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
    doneCount: ONBOARDING_COMPLETED_STEP_COUNT + steps.filter((step) => step.done).length,
    totalCount: ONBOARDING_COMPLETED_STEP_COUNT + steps.length,
    activeStep,
    title: activeStep.bannerTitle,
    description: activeStep.description,
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

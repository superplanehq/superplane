import { type PRFeedbackSourceId } from "./prFeedbackSettingsModel";

export type WorkspaceNextStepId = "pr-comments-handler" | "pr-checks-handler";

export type WorkspaceNextStepAction = { type: "configure-pr-feedback"; sourceId: PRFeedbackSourceId };

export interface WorkspaceNextStep {
  id: WorkspaceNextStepId;
  title: string;
  description: string;
  action: WorkspaceNextStepAction;
  done: boolean;
}

export interface WorkspaceNextStepContext {
  onboardingComplete: boolean;
  canConfigure: boolean;
  takenPRFeedbackSources: readonly PRFeedbackSourceId[];
}

export const WORKSPACE_NEXT_STEPS_COPY = {
  title: "You finished workspace setup",
  description:
    "The factory can implement tasks and will open pull requests for them. Configure status checks for those pull requests next.",
  configure: "Configure...",
} as const;

export function workspaceNextStepsProgressCopy(done: number, total: number): string {
  return `${done} of ${total} complete`;
}

const WORKSPACE_NEXT_STEPS: Array<
  Omit<WorkspaceNextStep, "done"> & { isDone: (ctx: WorkspaceNextStepContext) => boolean }
> = [
  {
    id: "pr-comments-handler",
    title: "Configure comments handler",
    description: "Start a fix when someone mentions the agent on a pull request.",
    action: { type: "configure-pr-feedback", sourceId: "discussion" },
    isDone: (ctx) => ctx.takenPRFeedbackSources.includes("discussion"),
  },
  {
    id: "pr-checks-handler",
    title: "Configure status checks handler",
    description: "Wait for status checks and start a fix when a selected check fails.",
    action: { type: "configure-pr-feedback", sourceId: "checks" },
    isDone: (ctx) => ctx.takenPRFeedbackSources.includes("checks"),
  },
];

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

export function runWorkspaceNextStepAction(
  action: WorkspaceNextStepAction,
  handlers: { configurePRFeedback: (sourceId: PRFeedbackSourceId) => void },
) {
  if (action.type === "configure-pr-feedback") {
    handlers.configurePRFeedback(action.sourceId);
  }
}

import type {
  CanvasesCanvasRunRef,
  FactoriesFactoryPrFeedbackHandler,
  FactoriesFactoryPullRequest,
} from "@/api-client";

import { isActiveCanvasRun } from "../lib/workOrderPullRequest";

export type PRFeedbackSettingsTab = "general" | "automation";

export function isPRFeedbackSettingsTab(value: string | null | undefined): value is PRFeedbackSettingsTab {
  return value === "general" || value === "automation";
}

export interface PRFeedbackDraftSettings {
  name: string;
  repository: string;
  mention: string;
  ignoreBots: boolean;
}

export const PR_FEEDBACK_SETTINGS_COPY = {
  tabsLabel: "PR feedback settings",
  generalTab: "General",
  automationTab: "Automation",
  nameLabel: "Name",
  nameHelper: "This name appears in the workspace app list.",
  repositoryLabel: "Repository",
  repositoryHelper: "Listen for mentions on pull requests in this repository.",
  mentionLabel: "Mention",
  mentionHelper: "Use an exact GitHub mention, for example @superplaneagent.",
  ignoreBotsLabel: "Ignore bot comments",
  ignoreBotsHelper: "Do not start a run when a bot writes the mention.",
  healthReady: "Ready",
  healthNeedsRepair: "Needs repair",
  healthReadyHelper: "This automation can receive a mention and address it.",
  healthNeedsRepairHelper: "Open the Automation tab and repair the canvas.",
  save: "Save",
  saving: "Saving",
  delete: "Delete automation",
  deleting: "Deleting",
  keep: "Keep automation",
  confirmDelete: "Delete this automation? This cannot be undone.",
  saveError: "Could not save PR feedback settings.",
  emptyTitle: "No PR feedback automation",
  emptyBody: "Create an automation that addresses pull request feedback when someone mentions the agent.",
  create: "Create automation",
  creating: "Creating",
  createError: "Could not create the PR feedback automation.",
  loading: "Loading PR feedback.",
  loadError: "Could not load PR feedback.",
  retry: "Retry",
  automationEmpty: "This automation has no canvas yet.",
  automationLoading: "Loading the canvas.",
  automationError: "Could not load the canvas.",
  retryAutomation: "Retry",
  editAutomation: "Edit automation",
} as const;

export function prFeedbackDraftFromHandler(handler: FactoriesFactoryPrFeedbackHandler): PRFeedbackDraftSettings {
  return {
    name: handler.name?.trim() || "Address PR feedback",
    repository: handler.settings?.subject?.repository?.trim() ?? "",
    mention: handler.settings?.discussion?.mention?.trim() || "@superplaneagent",
    ignoreBots: handler.settings?.discussion?.ignoreBots !== false,
  };
}

export const DEFAULT_PR_FEEDBACK_MENTION = "@superplaneagent";

/** The mention the handler watches, or the default before one is declared. */
export function prFeedbackMention(handler: FactoriesFactoryPrFeedbackHandler | undefined): string {
  return handler ? prFeedbackDraftFromHandler(handler).mention : DEFAULT_PR_FEEDBACK_MENTION;
}

/** Row title on the Verify lane. It says what the handler listens to. */
export function prFeedbackListenTitle(mention: string): string {
  return `Listening to ${mention} comments`;
}

export function normalizePRFeedbackDraft(draft: PRFeedbackDraftSettings): PRFeedbackDraftSettings {
  return {
    name: draft.name.trim(),
    repository: draft.repository.trim(),
    mention: draft.mention.trim(),
    ignoreBots: draft.ignoreBots,
  };
}

export function prFeedbackDraftIsValid(draft: PRFeedbackDraftSettings): boolean {
  const next = normalizePRFeedbackDraft(draft);
  return next.name.length > 0 && next.repository.length > 0 && next.mention.startsWith("@");
}

export function isActivePRFeedbackRun(run: CanvasesCanvasRunRef | undefined): boolean {
  return isActiveCanvasRun(run);
}

export function activePRFeedbackWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  const ids = new Set<string>();
  for (const pullRequest of pullRequests) {
    const workOrderId = pullRequest.workOrderId?.trim();
    if (!workOrderId) {
      continue;
    }
    if ((pullRequest.runs ?? []).some(isActiveCanvasRun)) {
      ids.add(workOrderId);
    }
  }
  return ids;
}

export type PRFeedbackLogRun = {
  canvasId: string;
  handlerName?: string;
  pullRequestNumber?: string;
  run: CanvasesCanvasRunRef;
};

export function oldestActivePRFeedbackRun(runs: CanvasesCanvasRunRef[]): CanvasesCanvasRunRef | undefined {
  const active = runs.filter((run) => isActiveCanvasRun(run));
  if (active.length === 0) {
    return undefined;
  }
  return [...active].sort((left, right) => Date.parse(left.createdAt ?? "") - Date.parse(right.createdAt ?? ""))[0];
}

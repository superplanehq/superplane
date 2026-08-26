import type { FactoriesFactoryPrFeedbackHandler, FactoriesFactoryPrFeedbackHandlerRun } from "@/api-client";
import { formatTimeAgo } from "@/lib/date";

import type { WorkOrderStatusNotePresentation } from "../lib/workOrderStatusNote";

export type PRFeedbackSettingsTab = "general" | "runs" | "automation";

export function isPRFeedbackSettingsTab(value: string | null | undefined): value is PRFeedbackSettingsTab {
  return value === "general" || value === "runs" || value === "automation";
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
  runsTab: "Runs",
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
  runsEmpty: "No runs yet. Mention the agent in a pull request comment or review to start a run.",
  runsLoading: "Loading runs.",
  runsError: "Could not load runs.",
  retryRuns: "Retry",
  automationEmpty: "This automation has no canvas yet.",
  automationLoading: "Loading the canvas.",
  automationError: "Could not load the canvas.",
  retryAutomation: "Retry",
  editAutomation: "Edit automation",
  viewRun: "Open the run",
  viewWorkOrder: "Open the work order",
  viewPullRequest: "Open the pull request",
  viewComment: "Open the comment",
  runWhen: "Started",
  runStatus: "Status",
  runTrigger: "Trigger",
} as const;

const RUN_STATUS_LABEL: Record<string, string> = {
  STATUS_QUEUED: "Queued",
  STATUS_RUNNING: "Running",
  STATUS_PASSED: "Passed",
  STATUS_FAILED: "Failed",
  STATUS_CANCELLED: "Cancelled",
};

const RUN_TRIGGER_LABEL: Record<string, string> = {
  TRIGGER_PR_COMMENT: "Comment",
  TRIGGER_PR_REVIEW: "Review",
  TRIGGER_PR_REVIEW_REPLY: "Reply",
};

export function prFeedbackDraftFromHandler(handler: FactoriesFactoryPrFeedbackHandler): PRFeedbackDraftSettings {
  return {
    name: handler.name?.trim() || "Address PR feedback",
    repository: handler.settings?.repository?.trim() ?? "",
    mention: handler.settings?.mention?.trim() || "@superplaneagent",
    ignoreBots: handler.settings?.ignoreBots !== false,
  };
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

export function isActivePRFeedbackRunStatus(status: FactoriesFactoryPrFeedbackHandlerRun["status"]): boolean {
  return status === "STATUS_QUEUED" || status === "STATUS_RUNNING";
}

export function oldestActivePRFeedbackRun(
  runs: FactoriesFactoryPrFeedbackHandlerRun[],
  workOrderId: string,
): FactoriesFactoryPrFeedbackHandlerRun | undefined {
  const active = runs.filter(
    (run) => Boolean(run.id) && run.workOrderId === workOrderId && isActivePRFeedbackRunStatus(run.status),
  );
  if (active.length === 0) {
    return undefined;
  }
  return [...active].sort(
    (left, right) => Date.parse(left.createdAt ?? "") - Date.parse(right.createdAt ?? ""),
  )[0];
}

export function addressingPRFeedbackNote(runHref: string): WorkOrderStatusNotePresentation {
  return {
    key: "pr-feedback-active",
    headline: "Addressing PR feedback",
    text: "SuperPlane is applying the current pull request feedback.",
    cta: { label: "Open the run", href: runHref },
  };
}

export function prFeedbackRunStatusLabel(status: FactoriesFactoryPrFeedbackHandlerRun["status"]): string {
  return (status && RUN_STATUS_LABEL[status]) || "Unknown";
}

export function prFeedbackRunTriggerLabel(trigger: FactoriesFactoryPrFeedbackHandlerRun["trigger"]): string {
  return (trigger && RUN_TRIGGER_LABEL[trigger]) || "Mention";
}

export function prFeedbackRunTimeLabel(run: FactoriesFactoryPrFeedbackHandlerRun): string {
  const value = run.startedAt ?? run.createdAt;
  if (!value) {
    return "—";
  }
  return formatTimeAgo(new Date(value));
}

export function prFeedbackRunTitle(run: FactoriesFactoryPrFeedbackHandlerRun): string {
  if (run.title?.trim()) {
    return run.title.trim();
  }
  if (run.pullRequestNumber) {
    return `Address feedback on PR #${run.pullRequestNumber}`;
  }
  return "Address PR feedback";
}

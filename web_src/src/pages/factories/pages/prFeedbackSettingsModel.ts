import type {
  CanvasesCanvasRunRef,
  FactoriesFactoryPrFeedbackHandler,
  FactoriesFactoryPrFeedbackHandlerSettings,
  FactoriesFactoryPrFeedbackHandlerSource,
  FactoriesFactoryPullRequest,
  FactoriesFactoryPullRequestActivity,
} from "@/api-client";
import githubIcon from "@/assets/icons/integrations/github.svg";

import { isActiveCanvasRun } from "../lib/workOrderPullRequest";

export type PRFeedbackSettingsTab = "general" | "automation";
export type PRFeedbackSourceId = "discussion" | "checks";

export function isPRFeedbackSettingsTab(value: string | null | undefined): value is PRFeedbackSettingsTab {
  return value === "general" || value === "automation";
}

export interface PRFeedbackSource {
  id: PRFeedbackSourceId;
  name: string;
  description: string;
  listenTitle: string;
  iconSrc: string;
  iconAlt: string;
  defaultName: string;
}

export const PR_FEEDBACK_SOURCES: PRFeedbackSource[] = [
  {
    id: "discussion",
    name: "Pull request discussion",
    description: "Address comments and reviews after a mention.",
    listenTitle: "Listening to pull request comments",
    iconSrc: githubIcon,
    iconAlt: "GitHub",
    defaultName: "Address PR feedback",
  },
  {
    id: "checks",
    name: "Pull request checks",
    description: "Wait for checks and start one agent run when selected checks fail.",
    listenTitle: "Monitoring pull request checks",
    iconSrc: githubIcon,
    iconAlt: "GitHub",
    defaultName: "Fix pull request checks",
  },
];

export function prFeedbackSourceById(id: string | undefined): PRFeedbackSource | undefined {
  return PR_FEEDBACK_SOURCES.find((source) => source.id === id);
}

export function prFeedbackSourceId(source?: FactoriesFactoryPrFeedbackHandlerSource): PRFeedbackSourceId {
  return source === "SOURCE_PULL_REQUEST_CHECKS" ? "checks" : "discussion";
}

export function apiPRFeedbackSource(sourceId: PRFeedbackSourceId): FactoriesFactoryPrFeedbackHandlerSource {
  return sourceId === "checks" ? "SOURCE_PULL_REQUEST_CHECKS" : "SOURCE_PULL_REQUEST_DISCUSSION";
}

export function takenPRFeedbackSourceIds(
  handlers: Array<{ source?: FactoriesFactoryPrFeedbackHandlerSource }>,
): PRFeedbackSourceId[] {
  return handlers.map((handler) => prFeedbackSourceId(handler.source));
}

export function hasAvailablePRFeedbackSource(taken: readonly PRFeedbackSourceId[]): boolean {
  return PR_FEEDBACK_SOURCES.some((source) => !taken.includes(source.id));
}

export interface PRFeedbackDraftSettings {
  source: PRFeedbackSourceId;
  name: string;
  repository: string;
  mention: string;
  ignoreBots: boolean;
  allowedBots: string[];
  checkNames: string[];
  maximumAttempts: number;
  runnerIntegrationIds: string[];
}

export const PR_FEEDBACK_SETTINGS_COPY = {
  tabsLabel: "PR feedback settings",
  generalTab: "General",
  automationTab: "Automation",
  nameLabel: "Name",
  nameHelper: "This name appears in the workspace app list.",
  repositoryLabel: "Repository",
  repositoryHelper: "Listen for mentions on pull requests in this repository.",
  checksRepositoryHelper: "Wait for checks on pull requests in this repository.",
  mentionLabel: "Mention",
  mentionHelper: "Use an exact GitHub mention, for example @superplaneagent.",
  ignoreBotsLabel: "Ignore bot comments",
  ignoreBotsHelper: "Do not start a run when a bot writes the mention.",
  allowedBotsLabel: "Allowed bots",
  allowedBotsHelper: "React to comments from these bots even without the mention. Use the bot login.",
  checkNamesLabel: "Status checks",
  checkNamesHelper:
    "SuperPlane waits for each selected check and fixes selected failures. Leave empty to monitor all checks.",
  checkNamesAdd: "Add",
  checkNamesPlaceholder: "lint",
  maximumAttemptsLabel: "Maximum automatic fix attempts",
  maximumAttemptsHelper:
    "SuperPlane pauses automatic fixes after this many consecutive attempts. Passing checks reset the count.",
  integrationsLabel: "Additional integration access",
  integrationsHelper: "Give the agent access to CI logs from other connected integrations.",
  integrationsEmpty: "No other connected integrations are available.",
  healthReady: "Ready",
  healthNeedsRepair: "Needs repair",
  healthReadyHelper: "This automation can receive a mention and address it.",
  healthChecksReadyHelper: "This automation can wait for checks and start a fix.",
  healthNeedsRepairHelper: "Open the Automation tab and repair the canvas.",
  save: "Save",
  saving: "Saving",
  delete: "Delete automation",
  deleting: "Deleting",
  keep: "Keep automation",
  confirmDelete: "Delete this automation? This cannot be undone.",
  saveError: "Could not save PR feedback settings.",
  emptyTitle: "No PR feedback automation",
  emptyBody: "Create an automation that addresses pull request feedback.",
  create: "Add feedback handler",
  creating: "Creating",
  createError: "Could not create the PR feedback automation.",
  addHandler: "Add feedback handler",
  pickerTitle: "Add feedback handler",
  pickerDescription: "Choose the signal that starts a pull request feedback run.",
  sourceTaken: "A handler for this source already exists.",
  loading: "Loading PR feedback.",
  loadError: "Could not load PR feedback.",
  retry: "Retry",
  automationEmpty: "This automation has no canvas yet.",
  automationLoading: "Loading the canvas.",
  automationError: "Could not load the canvas.",
  retryAutomation: "Retry",
  editAutomation: "Edit automation",
  waitingForAccess: "Waiting for another pull request activity",
} as const;

export function prFeedbackDraftFromHandler(handler: FactoriesFactoryPrFeedbackHandler): PRFeedbackDraftSettings {
  const source = prFeedbackSourceId(handler.source);
  const discussion = handler.settings?.discussion;
  const checks = handler.settings?.checks;
  return {
    source,
    name: handler.name?.trim() || prFeedbackSourceById(source)?.defaultName || "Address PR feedback",
    repository: handler.settings?.subject?.repository?.trim() ?? "",
    mention: discussion?.mention?.trim() || "@superplaneagent",
    ignoreBots: discussion?.ignoreBots !== false,
    allowedBots: discussion?.allowedBots ?? [],
    checkNames: checks?.names ?? [],
    maximumAttempts: checks?.maximumAttempts ?? 3,
    runnerIntegrationIds: checks?.runnerIntegrationIds ?? [],
  };
}

export function prFeedbackListenTitle(source?: FactoriesFactoryPrFeedbackHandlerSource | PRFeedbackSourceId): string {
  const sourceId = source === "discussion" || source === "checks" ? source : prFeedbackSourceId(source);
  return prFeedbackSourceById(sourceId)?.listenTitle ?? "Listening to pull request comments";
}

export function normalizePRFeedbackDraft(draft: PRFeedbackDraftSettings): PRFeedbackDraftSettings {
  return {
    source: draft.source,
    name: draft.name.trim(),
    repository: draft.repository.trim(),
    mention: draft.mention.trim(),
    ignoreBots: draft.ignoreBots,
    allowedBots: normalizeAllowedBots(draft.allowedBots),
    checkNames: normalizeUniqueStrings(draft.checkNames),
    maximumAttempts: draft.maximumAttempts,
    runnerIntegrationIds: normalizeUniqueStrings(draft.runnerIntegrationIds),
  };
}

/** Add one exact name. Keep commas inside the name. Skip blanks and duplicates. */
export function appendUniqueTrimmedString(items: string[], raw: string): string[] {
  const name = raw.trim();
  if (name.length === 0 || items.includes(name)) {
    return items;
  }
  return [...items, name];
}

function normalizeAllowedBots(bots: string[]): string[] {
  const seen = new Set<string>();
  const normalized: string[] = [];
  for (const bot of bots) {
    const trimmed = bot.trim().replace(/^@/, "");
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    normalized.push(trimmed);
  }
  return normalized;
}

function normalizeUniqueStrings(values: string[]): string[] {
  const seen = new Set<string>();
  const normalized: string[] = [];
  for (const value of values) {
    const trimmed = value.trim();
    if (!trimmed) {
      continue;
    }
    const key = trimmed.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    normalized.push(trimmed);
  }
  return normalized;
}

export function prFeedbackDraftIsValid(draft: PRFeedbackDraftSettings): boolean {
  const next = normalizePRFeedbackDraft(draft);
  if (next.name.length === 0 || next.repository.length === 0) {
    return false;
  }
  if (next.source === "checks") {
    return next.maximumAttempts >= 1 && next.maximumAttempts <= 10;
  }
  return next.mention.startsWith("@");
}

export function prFeedbackSettingsToApi(draft: PRFeedbackDraftSettings): FactoriesFactoryPrFeedbackHandlerSettings {
  if (draft.source === "checks") {
    return {
      subject: { repository: draft.repository },
      checks: {
        names: draft.checkNames,
        maximumAttempts: draft.maximumAttempts,
        runnerIntegrationIds: draft.runnerIntegrationIds,
      },
    };
  }
  return {
    subject: { repository: draft.repository },
    discussion: {
      mention: draft.mention,
      ignoreBots: draft.ignoreBots,
      allowedBots: draft.allowedBots,
    },
  };
}

export function isActivePRFeedbackRun(run: CanvasesCanvasRunRef | undefined): boolean {
  return isActiveCanvasRun(run);
}

export function isActivePRFeedbackActivity(activity: FactoriesFactoryPullRequestActivity | undefined): boolean {
  return activity?.state === "active" && isActiveCanvasRun(activity.run);
}

const WAITING_ON_CHECKS_DESCRIPTION = /^Waiting for checks\b/i;
const CHECKS_PASSED_DESCRIPTION = /^Checks passed\b/i;

/** Concurrent check-wait: SuperPlane watches CI and does not address comments yet. */
export function isWaitingOnChecksActivity(activity: FactoriesFactoryPullRequestActivity | undefined): boolean {
  if (!activity || !isActivePRFeedbackActivity(activity)) {
    return false;
  }
  if (activity.access === "concurrent") {
    return true;
  }
  if (activity.access === "exclusive" || activity.access === "waiting") {
    return false;
  }
  return WAITING_ON_CHECKS_DESCRIPTION.test(activity.description ?? "");
}

export function isAddressingFeedbackActivity(activity: FactoriesFactoryPullRequestActivity | undefined): boolean {
  return isActivePRFeedbackActivity(activity) && !isWaitingOnChecksActivity(activity);
}

export function addressingFeedbackWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  return new Set(addressingFeedbackLabelsByWorkOrder(pullRequests).keys());
}

const FIXING_CHECKS_DESCRIPTION = /^Fixing failed checks\b/i;

export function addressingFeedbackLabelsByWorkOrder(
  pullRequests: FactoriesFactoryPullRequest[],
): ReadonlyMap<string, string> {
  const labels = new Map<string, string>();
  for (const pullRequest of pullRequests) {
    const workOrderId = pullRequest.workOrderId?.trim();
    if (!workOrderId) {
      continue;
    }
    const activity = latestAddressingActivity(pullRequest);
    if (activity) {
      labels.set(workOrderId, addressingFeedbackCardLabel(activity));
      continue;
    }
    if (
      (pullRequest.activities ?? []).length === 0 &&
      (pullRequest.runs ?? []).some((linked) => isActiveCanvasRun(linked.run))
    ) {
      labels.set(workOrderId, "Addressing user feedback");
    }
  }
  return labels;
}

function latestAddressingActivity(
  pullRequest: FactoriesFactoryPullRequest,
): FactoriesFactoryPullRequestActivity | undefined {
  const active = (pullRequest.activities ?? []).filter((activity) => isAddressingFeedbackActivity(activity));
  if (active.length === 0) {
    return undefined;
  }
  return [...active].sort(
    (left, right) => Date.parse(right.run?.createdAt ?? "") - Date.parse(left.run?.createdAt ?? ""),
  )[0];
}

function addressingFeedbackCardLabel(activity: FactoriesFactoryPullRequestActivity): string {
  const description = activity.description?.trim() ?? "";
  if (activity.revision || FIXING_CHECKS_DESCRIPTION.test(description)) {
    return prFeedbackActivityLabel(activity);
  }
  return "Addressing user feedback";
}

export function waitingOnChecksWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  return prFeedbackWorkOrderIds(pullRequests, "checks-wait");
}

export function isChecksPassedActivity(activity: FactoriesFactoryPullRequestActivity | undefined): boolean {
  if (!activity || isActivePRFeedbackActivity(activity)) {
    return false;
  }
  return CHECKS_PASSED_DESCRIPTION.test(activity.description ?? "");
}

/** Latest finished passed check wait. Active waits and repairs win. */
export function checksPassedWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  const waiting = waitingOnChecksWorkOrderIds(pullRequests);
  const addressing = addressingFeedbackWorkOrderIds(pullRequests);
  const ids = new Set<string>();
  for (const pullRequest of pullRequests) {
    const workOrderId = pullRequest.workOrderId?.trim();
    if (!workOrderId || waiting.has(workOrderId) || addressing.has(workOrderId)) {
      continue;
    }
    if (isChecksPassedActivity(latestCheckActivity(pullRequest.activities ?? []))) {
      ids.add(workOrderId);
    }
  }
  return ids;
}

function latestCheckActivity(
  activities: FactoriesFactoryPullRequestActivity[],
): FactoriesFactoryPullRequestActivity | undefined {
  const related = activities.filter((activity) => isCheckRelatedActivity(activity));
  if (related.length === 0) {
    return undefined;
  }
  return [...related].sort(
    (left, right) => Date.parse(right.run?.createdAt ?? "") - Date.parse(left.run?.createdAt ?? ""),
  )[0];
}

function isCheckRelatedActivity(activity: FactoriesFactoryPullRequestActivity): boolean {
  if (activity.revision) {
    return true;
  }
  const description = activity.description ?? "";
  return (
    WAITING_ON_CHECKS_DESCRIPTION.test(description) ||
    CHECKS_PASSED_DESCRIPTION.test(description) ||
    FIXING_CHECKS_DESCRIPTION.test(description)
  );
}

/** Tasks with an active discussion or exclusive-repair run. */
export function activePRFeedbackWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  return addressingFeedbackWorkOrderIds(pullRequests);
}

function prFeedbackWorkOrderIds(
  pullRequests: FactoriesFactoryPullRequest[],
  kind: "addressing" | "checks-wait",
): ReadonlySet<string> {
  const ids = new Set<string>();
  for (const pullRequest of pullRequests) {
    const workOrderId = pullRequest.workOrderId?.trim();
    if (!workOrderId) {
      continue;
    }
    const activities = pullRequest.activities ?? [];
    if (activities.length > 0) {
      const matches =
        kind === "checks-wait"
          ? activities.some((activity) => isWaitingOnChecksActivity(activity))
          : activities.some((activity) => isAddressingFeedbackActivity(activity));
      if (matches) {
        ids.add(workOrderId);
      }
      continue;
    }
    if (kind === "addressing" && (pullRequest.runs ?? []).some((linked) => isActiveCanvasRun(linked.run))) {
      ids.add(workOrderId);
    }
  }
  return ids;
}

export function prFeedbackActivityLabel(activity: FactoriesFactoryPullRequestActivity): string {
  if (activity.state === "limit_reached") {
    const limit = activity.attemptLimit ?? activity.attempt ?? 3;
    return activity.description?.trim() || `Automatic fixes paused after ${limit} attempts`;
  }
  if (activity.access === "waiting") {
    return PR_FEEDBACK_SETTINGS_COPY.waitingForAccess;
  }
  return activity.description?.trim() || "Pull request activity";
}

export function prFeedbackActivityAttemptLabel(activity: FactoriesFactoryPullRequestActivity): string | undefined {
  if (!activity.attempt || activity.attempt < 1) {
    return undefined;
  }
  const limit = activity.attemptLimit && activity.attemptLimit > 0 ? activity.attemptLimit : 3;
  return `Attempt ${activity.attempt} of ${limit}`;
}

export type PRFeedbackActivityKind = "checks-wait" | "addressing";

export type PRFeedbackLogRun = {
  canvasId: string;
  handlerName?: string;
  pullRequestNumber?: string;
  description?: string;
  attemptLabel?: string;
  costCents?: string;
  totalTokens?: string;
  kind?: PRFeedbackActivityKind;
  run: CanvasesCanvasRunRef;
};

export function prFeedbackActivityKind(activity: FactoriesFactoryPullRequestActivity): PRFeedbackActivityKind {
  return isWaitingOnChecksActivity(activity) ? "checks-wait" : "addressing";
}

export function oldestActivePRFeedbackRun(runs: CanvasesCanvasRunRef[]): CanvasesCanvasRunRef | undefined {
  const active = runs.filter((run) => isActiveCanvasRun(run));
  if (active.length === 0) {
    return undefined;
  }
  return [...active].sort((left, right) => Date.parse(left.createdAt ?? "") - Date.parse(right.createdAt ?? ""))[0];
}

import type {
  CanvasesCanvasRunRef,
  FactoriesFactoryPullRequest,
  FactoriesFactoryPullRequestActivity,
} from "@/api-client";

import { isActiveCanvasRun } from "../lib/workOrderPullRequest";

const WAITING_FOR_ACCESS_LABEL = "Waiting for another pull request activity";

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
  const paused = fixesPausedWorkOrderIds(pullRequests);
  const ids = new Set<string>();
  for (const pullRequest of pullRequests) {
    const workOrderId = pullRequest.workOrderId?.trim();
    if (!workOrderId || waiting.has(workOrderId) || addressing.has(workOrderId) || paused.has(workOrderId)) {
      continue;
    }
    if (isChecksPassedActivity(latestCheckActivity(pullRequest.activities ?? []))) {
      ids.add(workOrderId);
    }
  }
  return ids;
}

export function isFixesPausedActivity(activity: FactoriesFactoryPullRequestActivity | undefined): boolean {
  return activity?.state === "limit_reached";
}

/** Latest check activity stopped because the attempt limit was reached. */
export function fixesPausedWorkOrderIds(pullRequests: FactoriesFactoryPullRequest[]): ReadonlySet<string> {
  const waiting = waitingOnChecksWorkOrderIds(pullRequests);
  const addressing = addressingFeedbackWorkOrderIds(pullRequests);
  const ids = new Set<string>();
  for (const pullRequest of pullRequests) {
    const workOrderId = pullRequest.workOrderId?.trim();
    if (!workOrderId || waiting.has(workOrderId) || addressing.has(workOrderId)) {
      continue;
    }
    if (isFixesPausedActivity(latestCheckActivity(pullRequest.activities ?? []))) {
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
  if (activity.state === "limit_reached" || activity.revision) {
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
    return (
      activity.description?.trim() || `Automatic fixes paused after ${limit} ${limit === 1 ? "attempt" : "attempts"}`
    );
  }
  if (activity.access === "waiting") {
    return WAITING_FOR_ACCESS_LABEL;
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

export type PRFeedbackActivityKind = "checks-wait" | "addressing" | "fixes-paused";

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
  if (isFixesPausedActivity(activity)) {
    return "fixes-paused";
  }
  return isWaitingOnChecksActivity(activity) ? "checks-wait" : "addressing";
}

export function oldestActivePRFeedbackRun(runs: CanvasesCanvasRunRef[]): CanvasesCanvasRunRef | undefined {
  const active = runs.filter((run) => isActiveCanvasRun(run));
  if (active.length === 0) {
    return undefined;
  }
  return [...active].sort((left, right) => Date.parse(left.createdAt ?? "") - Date.parse(right.createdAt ?? ""))[0];
}

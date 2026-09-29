import type { FactoriesFactoryPullRequest } from "@/api-client";

import type { SplitRunPhase } from "./splitRunMocks";

export interface PullRequestActivityGroup {
  id: string;
  pullRequest?: FactoriesFactoryPullRequest;
  phases: SplitRunPhase[];
}

export interface SplitRunActivityGroups {
  taskAutomationPhases: SplitRunPhase[];
  pullRequestActivityGroups: PullRequestActivityGroup[];
}

export function groupSplitRunActivities(phases: SplitRunPhase[]): SplitRunActivityGroups {
  const taskAutomationPhases: SplitRunPhase[] = [];
  const pullRequestActivityPhases: Array<{ phase: SplitRunPhase; originalIndex: number }> = [];

  phases.forEach((phase, originalIndex) => {
    if (phase.pullRequestActivity) {
      pullRequestActivityPhases.push({ phase, originalIndex });
      return;
    }
    taskAutomationPhases.push(phase);
  });

  pullRequestActivityPhases.sort((left, right) => {
    const timeDifference = activityTimestamp(left.phase) - activityTimestamp(right.phase);
    return timeDifference || left.originalIndex - right.originalIndex;
  });

  const groupsByPullRequest = new Map<string, PullRequestActivityGroup>();
  for (const { phase } of pullRequestActivityPhases) {
    const activity = phase.pullRequestActivity;
    if (!activity) {
      continue;
    }
    const pullRequestId = pullRequestGroupKey(activity.pullRequest);
    const existing = groupsByPullRequest.get(pullRequestId);
    if (existing) {
      existing.phases.push(phase);
      continue;
    }
    groupsByPullRequest.set(pullRequestId, {
      id: pullRequestId,
      pullRequest: activity.pullRequest,
      phases: [phase],
    });
  }

  return {
    taskAutomationPhases,
    pullRequestActivityGroups: [...groupsByPullRequest.values()],
  };
}

function activityTimestamp(phase: SplitRunPhase): number {
  const timestamp = Date.parse(phase.pullRequestActivity?.startedAt ?? "");
  return Number.isFinite(timestamp) ? timestamp : Number.POSITIVE_INFINITY;
}

function pullRequestGroupKey(pullRequest?: FactoriesFactoryPullRequest): string {
  if (pullRequest?.id?.trim()) {
    return pullRequest.id.trim();
  }
  if (pullRequest?.url?.trim()) {
    return pullRequest.url.trim();
  }
  const repository = pullRequest?.repository?.trim();
  const number = pullRequest?.number?.trim();
  return repository || number ? `${repository ?? ""}#${number ?? ""}` : "pull-request";
}

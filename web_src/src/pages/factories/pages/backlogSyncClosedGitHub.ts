import type { FactoriesFactoryIntake } from "@/api-client";

export const BACKLOG_SYNC_GITHUB_COPY = {
  menu: "Sync closed GitHub issues",
  none: "No tasks to close.",
  closedOne: "Closed 1 task.",
  failed: "SuperPlane could not check GitHub. Try again.",
  failedOne: "SuperPlane could not check 1 GitHub issue.",
} as const;

export function backlogSyncClosedGitHubClosedMessage(count: number): string {
  if (count === 1) {
    return BACKLOG_SYNC_GITHUB_COPY.closedOne;
  }
  return `Closed ${count} tasks.`;
}

export function backlogSyncClosedGitHubFailedMessage(count: number): string {
  if (count === 1) {
    return BACKLOG_SYNC_GITHUB_COPY.failedOne;
  }
  return `SuperPlane could not check ${count} GitHub issues.`;
}

export function hasGitHubIssuesIntake(intakes: FactoriesFactoryIntake[] | undefined): boolean {
  return Boolean(intakes?.some((intake) => intake.source === "SOURCE_GITHUB_ISSUES"));
}

export type SyncClosedGitHubBacklogResult = {
  closedCount: number;
  failedCount: number;
};

export function syncClosedGitHubToast(result: SyncClosedGitHubBacklogResult): {
  kind: "success" | "error" | "info";
  message: string;
} {
  if (result.failedCount > 0 && result.closedCount === 0) {
    return { kind: "error", message: BACKLOG_SYNC_GITHUB_COPY.failed };
  }

  const parts: string[] = [];
  if (result.closedCount === 0) {
    parts.push(BACKLOG_SYNC_GITHUB_COPY.none);
  } else {
    parts.push(backlogSyncClosedGitHubClosedMessage(result.closedCount));
  }
  if (result.failedCount > 0) {
    parts.push(backlogSyncClosedGitHubFailedMessage(result.failedCount));
  }

  return {
    kind: result.failedCount > 0 ? "error" : result.closedCount > 0 ? "success" : "info",
    message: parts.join(" "),
  };
}

import type { FactoriesFactoryIntake } from "@/api-client";
import type { RefreshBacklogResult } from "@/hooks/useFactoryIntakeData";

const REFRESHABLE_INTAKE_SOURCES: ReadonlySet<FactoriesFactoryIntake["source"]> = new Set([
  "SOURCE_GITHUB_ISSUES",
  "SOURCE_JIRA_ISSUES",
  "SOURCE_PRODUCTIVE_TASKS",
]);

export const BACKLOG_REFRESH_COPY = {
  menu: "Refresh backlog",
  current: "Backlog is up to date.",
  archivedOne: "Archived 1 task.",
  failed: "SuperPlane could not refresh the backlog. Try again.",
  failedItemOne: "SuperPlane could not check 1 intake item.",
  failedSourceOne: "SuperPlane could not connect to 1 intake source.",
} as const;

export function hasRefreshableIntake(intakes: FactoriesFactoryIntake[] | undefined): boolean {
  return Boolean(intakes?.some((intake) => intake.source != null && REFRESHABLE_INTAKE_SOURCES.has(intake.source)));
}

export function canRefreshBacklog(
  intakes: FactoriesFactoryIntake[] | undefined,
  canUpdateWorkOrders: boolean,
): boolean {
  return canUpdateWorkOrders && hasRefreshableIntake(intakes);
}

export function backlogRefreshToast(result: RefreshBacklogResult): {
  kind: "success" | "error" | "info";
  message: string;
} {
  const failureCount = result.failedItemCount + result.failedSourceCount;
  if (result.archivedCount === 0 && failureCount === 0) {
    return { kind: "info", message: BACKLOG_REFRESH_COPY.current };
  }

  const parts: string[] = [];
  if (result.archivedCount > 0) {
    parts.push(archivedMessage(result.archivedCount));
  }
  if (result.failedItemCount > 0) {
    parts.push(failedItemMessage(result.failedItemCount));
  }
  if (result.failedSourceCount > 0) {
    parts.push(failedSourceMessage(result.failedSourceCount));
  }

  return {
    kind: failureCount > 0 ? "error" : "success",
    message: parts.join(" "),
  };
}

function archivedMessage(count: number): string {
  return count === 1 ? BACKLOG_REFRESH_COPY.archivedOne : `Archived ${count} tasks.`;
}

function failedItemMessage(count: number): string {
  return count === 1 ? BACKLOG_REFRESH_COPY.failedItemOne : `SuperPlane could not check ${count} intake items.`;
}

function failedSourceMessage(count: number): string {
  return count === 1
    ? BACKLOG_REFRESH_COPY.failedSourceOne
    : `SuperPlane could not connect to ${count} intake sources.`;
}

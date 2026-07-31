export type DraftBranchEditStatus = "uncommitted" | "ready" | "no-changes";

export function resolveDraftBranchEditStatus(
  hasUncommittedChanges: boolean,
  hasPublishableChanges: boolean,
): DraftBranchEditStatus {
  if (hasUncommittedChanges) {
    return "uncommitted";
  }

  if (hasPublishableChanges) {
    return "ready";
  }

  return "no-changes";
}

export function draftBranchRowBackgroundClassName(isActive: boolean, editStatus: DraftBranchEditStatus): string {
  if (!isActive) {
    return "bg-surface-subtle";
  }

  if (editStatus === "uncommitted") {
    return "bg-status-warning-subtle";
  }

  return "bg-status-info-subtle";
}

const grayStatusBadgeClassName =
  "rounded bg-action-neutral px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-content-secondary";
const activeBlueStatusBadgeClassName =
  "rounded bg-blue-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-blue-800 dark:bg-blue-400 dark:text-blue-950";
const activeOrangeStatusBadgeClassName =
  "rounded bg-orange-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-orange-800 dark:bg-orange-400 dark:text-orange-950";

export function draftBranchStatusBadge(editStatus: DraftBranchEditStatus, isActive: boolean) {
  if (editStatus === "uncommitted") {
    return {
      label: "Uncommitted changes",
      className: isActive ? activeOrangeStatusBadgeClassName : grayStatusBadgeClassName,
    };
  }

  if (editStatus === "no-changes") {
    return {
      label: "No changes",
      className: grayStatusBadgeClassName,
    };
  }

  return {
    label: "Ready to publish",
    className: isActive ? activeBlueStatusBadgeClassName : grayStatusBadgeClassName,
  };
}

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
    return "bg-slate-50 dark:bg-gray-900";
  }

  if (editStatus === "uncommitted") {
    return "bg-orange-50 dark:bg-gray-800";
  }

  return "bg-blue-50 dark:bg-gray-800";
}

const grayStatusBadgeClassName =
  "rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-600 dark:bg-gray-800 dark:text-gray-400";
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

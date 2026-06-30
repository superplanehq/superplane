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
    return "bg-slate-50";
  }

  if (editStatus === "uncommitted") {
    return "bg-orange-50";
  }

  return "bg-blue-50";
}

const grayStatusBadgeClassName =
  "rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-slate-600";
const activeBlueStatusBadgeClassName =
  "rounded bg-blue-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-blue-800";
const activeOrangeStatusBadgeClassName =
  "rounded bg-orange-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-orange-800";

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

export type DraftEditTabTone = "uncommitted" | "ready" | "neutral";

export function draftEditTabToneFromStaging(hasUncommittedChanges: boolean, isEditing: boolean): DraftEditTabTone {
  if (!isEditing || !hasUncommittedChanges) {
    return "neutral";
  }

  return "uncommitted";
}

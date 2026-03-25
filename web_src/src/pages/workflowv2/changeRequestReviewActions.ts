import type { CanvasesCanvasChangeRequest, CanvasesCanvasChangeRequestApprovalConfig } from "@/api-client";

/** Matches list / detail views: substring checks on API status strings. */
export function normalizeChangeRequestStatus(status?: string): "open" | "published" | "rejected" | "unknown" {
  const value = (status || "").toLowerCase();
  if (value.includes("open")) return "open";
  if (value.includes("publish")) return "published";
  if (value.includes("reject")) return "rejected";
  return "unknown";
}

export function normalizeApprovalState(state?: string): "approved" | "rejected" | "unapproved" | "unknown" {
  const value = (state || "").toLowerCase();
  if (value.includes("unapproved")) return "unapproved";
  if (value.includes("approved")) return "approved";
  if (value.includes("rejected")) return "rejected";
  return "unknown";
}

export function isChangeRequestConflicted(changeRequest?: CanvasesCanvasChangeRequest): boolean {
  if (!changeRequest) {
    return false;
  }

  if (typeof changeRequest.metadata?.isConflicted === "boolean") {
    return changeRequest.metadata.isConflicted;
  }

  return (changeRequest.diff?.conflictingNodeIds || []).length > 0;
}

/**
 * Same rules as version sidebar rows and the version node diff dialog
 * (approve / unapprove / publish / reject / reopen / resolve).
 */
export function getChangeRequestReviewActionFlags(input: {
  changeRequest: CanvasesCanvasChangeRequest | undefined;
  changeRequestApprovalConfig: CanvasesCanvasChangeRequestApprovalConfig | undefined;
  canUpdateCanvas: boolean;
  currentUserId?: string;
}) {
  const { changeRequest, changeRequestApprovalConfig, canUpdateCanvas, currentUserId } = input;
  const itemStatus = normalizeChangeRequestStatus(changeRequest?.metadata?.status);
  const hasConflicts = isChangeRequestConflicted(changeRequest);
  const itemApprovals = changeRequest?.approvals || [];
  const itemActiveApprovedCount = itemApprovals.filter(
    (approval) => normalizeApprovalState(approval.state) === "approved" && !approval.invalidatedAt,
  ).length;
  const itemRequiredApprovalsCount =
    (changeRequestApprovalConfig?.items?.length || 0) > 0 ? changeRequestApprovalConfig?.items?.length || 0 : 1;
  const itemActiveApprovals = itemApprovals.filter((approval) => !approval.invalidatedAt);
  const itemHasCurrentUserActiveApproval = currentUserId
    ? itemActiveApprovals.some(
        (approval) => normalizeApprovalState(approval.state) === "approved" && approval.actor?.id === currentUserId,
      )
    : false;
  const itemApprovalRequirementsSatisfied = itemActiveApprovedCount >= itemRequiredApprovalsCount;
  const itemCanApprove = canUpdateCanvas && itemStatus === "open" && !hasConflicts && !itemHasCurrentUserActiveApproval;
  const itemCanUnapprove = canUpdateCanvas && itemStatus === "open" && itemHasCurrentUserActiveApproval;
  const itemCanPublish = canUpdateCanvas && itemStatus === "open" && !hasConflicts && itemApprovalRequirementsSatisfied;
  const itemCanReject = canUpdateCanvas && itemStatus === "open";
  const itemCanReopen = canUpdateCanvas && itemStatus === "rejected";
  const itemCanResolveConflicts =
    canUpdateCanvas &&
    itemStatus === "open" &&
    hasConflicts &&
    !!changeRequest?.version?.spec?.nodes &&
    !!changeRequest?.version?.spec?.edges;

  return {
    itemStatus,
    hasConflicts,
    itemActiveApprovedCount,
    itemRequiredApprovalsCount,
    itemCanApprove,
    itemCanUnapprove,
    itemCanPublish,
    itemCanReject,
    itemCanReopen,
    itemCanResolveConflicts,
  };
}

export type ChangeRequestReviewPhase =
  | { kind: "none" }
  | {
      kind: "awaiting";
      label: string;
      dotClassName: string;
      labelClassName: string;
      sidebarRowActiveClassName: string;
      floatingBarBgClassName: string;
      floatingBarDotClassName: string;
      floatingBarTitleClassName: string;
    }
  | {
      kind: "approved-not-published";
      label: string;
      dotClassName: string;
      labelClassName: string;
      sidebarRowActiveClassName: string;
      floatingBarBgClassName: string;
      floatingBarDotClassName: string;
      floatingBarTitleClassName: string;
    };

/**
 * UI label + colors for open change requests (sidebar row, floating bar, dialog subtitle).
 * `canUpdateCanvas` is not used — only approval counts and conflicts matter.
 */
export function getChangeRequestReviewPhase(
  changeRequest: CanvasesCanvasChangeRequest | undefined,
  changeRequestApprovalConfig: CanvasesCanvasChangeRequestApprovalConfig | undefined,
): ChangeRequestReviewPhase {
  if (!changeRequest) {
    return { kind: "none" };
  }

  const flags = getChangeRequestReviewActionFlags({
    changeRequest,
    changeRequestApprovalConfig,
    canUpdateCanvas: true,
    currentUserId: undefined,
  });

  if (flags.itemStatus !== "open") {
    return { kind: "none" };
  }

  const isApprovedNotPublished =
    !flags.hasConflicts && flags.itemActiveApprovedCount >= flags.itemRequiredApprovalsCount;

  if (isApprovedNotPublished) {
    return {
      kind: "approved-not-published",
      label: "Approved, not Published",
      dotClassName: "text-[9px] text-yellow-600",
      labelClassName: "font-medium text-yellow-600",
      sidebarRowActiveClassName: "bg-yellow-100",
      floatingBarBgClassName: "bg-yellow-50",
      floatingBarDotClassName: "text-[11px] text-yellow-600",
      floatingBarTitleClassName: "text-yellow-600",
    };
  }

  return {
    kind: "awaiting",
    label: "Awaiting Approval",
    dotClassName: "text-[9px] text-orange-500",
    labelClassName: "font-medium text-orange-500",
    sidebarRowActiveClassName: "bg-orange-100",
    floatingBarBgClassName: "bg-orange-50",
    floatingBarDotClassName: "text-[11px] text-orange-500",
    floatingBarTitleClassName: "text-orange-500",
  };
}

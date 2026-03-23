import {
  CanvasesCanvasChangeRequest,
  CanvasesCanvasChangeRequestApprovalConfig,
  CanvasesCanvasVersion,
} from "@/api-client";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { useMemo } from "react";
import { formatTimestamp, summarizeNodeDiff, VersionNodeDiffAccordion } from "./VersionNodeDiff";
import { getChangeRequestReviewActionFlags, getChangeRequestReviewPhase } from "./changeRequestReviewActions";

export type CanvasVersionNodeDiffContext = {
  version: CanvasesCanvasVersion;
  previousVersion: CanvasesCanvasVersion;
  changeRequest?: CanvasesCanvasChangeRequest;
};

export function CanvasVersionNodeDiffDialog({
  context,
  onOpenChange,
  liveVersionOwnerProfilesById,
  changeRequestApprovalConfig,
  canActOnChangeRequests,
  currentUserId,
  changeRequestActionPending,
  onApproveChangeRequest,
  onUnapproveChangeRequest,
  onPublishChangeRequest,
  onRejectChangeRequest,
  onReopenChangeRequest,
  liveChangeRequest,
  resolvePending,
  onGoToVersioningToResolveConflicts,
}: {
  context: CanvasVersionNodeDiffContext | null;
  onOpenChange: (open: boolean) => void;
  liveVersionOwnerProfilesById?: Map<string, { name: string; avatarUrl?: string }>;
  changeRequestApprovalConfig?: CanvasesCanvasChangeRequestApprovalConfig;
  canActOnChangeRequests?: boolean;
  currentUserId?: string;
  changeRequestActionPending?: boolean;
  onApproveChangeRequest?: (changeRequestId: string) => void;
  onUnapproveChangeRequest?: (changeRequestId: string) => void;
  onPublishChangeRequest?: (changeRequestId: string) => void;
  onRejectChangeRequest?: (changeRequestId: string) => void;
  onReopenChangeRequest?: (changeRequestId: string) => void;
  /** Fresh row from change-request query so actions update after approve/unapprove/publish. */
  liveChangeRequest?: CanvasesCanvasChangeRequest;
  resolvePending?: boolean;
  /** Open the conflict resolver (full-screen) for this change request. */
  onGoToVersioningToResolveConflicts?: (changeRequestId: string) => void;
}) {
  const effectiveChangeRequest = liveChangeRequest ?? context?.changeRequest;
  const changeRequestId = effectiveChangeRequest?.metadata?.id || "";

  const diffSummary = useMemo(() => {
    if (!context) {
      return null;
    }
    return summarizeNodeDiff(context.version, context.previousVersion);
  }, [context]);
  const diffOwner = useMemo(() => {
    const changeRequestOwner = effectiveChangeRequest?.metadata?.owner;
    if (!changeRequestOwner) {
      return null;
    }

    const profile = changeRequestOwner.id ? liveVersionOwnerProfilesById?.get(changeRequestOwner.id) : undefined;
    const name = changeRequestOwner.name || profile?.name || "Unknown user";

    return {
      name,
      avatarUrl: profile?.avatarUrl,
    };
  }, [effectiveChangeRequest, liveVersionOwnerProfilesById]);
  const diffChangedCount = effectiveChangeRequest?.diff?.changedNodeIds?.length || 0;
  const diffConflictCount = effectiveChangeRequest?.diff?.conflictingNodeIds?.length || 0;

  const reviewFlags = useMemo(
    () =>
      getChangeRequestReviewActionFlags({
        changeRequest: effectiveChangeRequest,
        changeRequestApprovalConfig,
        canUpdateCanvas: !!canActOnChangeRequests,
        currentUserId,
      }),
    [effectiveChangeRequest, changeRequestApprovalConfig, canActOnChangeRequests, currentUserId],
  );

  const {
    itemStatus,
    hasConflicts: diffHasConflicts,
    itemActiveApprovedCount: diffActiveApprovedCount,
    itemRequiredApprovalsCount: diffRequiredApprovalsCount,
    itemCanApprove: canApprove,
    itemCanUnapprove: canUnapprove,
    itemCanPublish: canPublish,
    itemCanReject: canReject,
    itemCanReopen: canReopen,
    itemCanResolveConflicts: canResolveConflicts,
  } = reviewFlags;

  const reviewPhase = useMemo(
    () => getChangeRequestReviewPhase(effectiveChangeRequest, changeRequestApprovalConfig),
    [effectiveChangeRequest, changeRequestApprovalConfig],
  );

  const showReviewActionsSection = !!effectiveChangeRequest && !!changeRequestId && itemStatus !== "published";

  const diffActivityItems = useMemo(() => {
    if (!effectiveChangeRequest) {
      return [];
    }

    const approvals = effectiveChangeRequest.approvals || [];
    const ownerRef = effectiveChangeRequest.metadata?.owner;
    const ownerProfile = ownerRef?.id ? liveVersionOwnerProfilesById?.get(ownerRef.id) : undefined;
    const ownerName = ownerRef?.name || ownerProfile?.name || "Unknown user";
    const items: Array<{
      id: string;
      title: string;
      detail: string;
      timestamp: string;
      tone: "slate" | "emerald" | "rose" | "amber";
      invalidated?: boolean;
      actorName?: string;
    }> = [];
    items.push({
      id: "opened",
      title: "Opened",
      detail: "opened this change request.",
      timestamp: formatTimestamp(effectiveChangeRequest.metadata?.createdAt) || "unknown time",
      tone: "slate",
      actorName: ownerName,
    });

    approvals.forEach((approval, index) => {
      const value = (approval.state || "").toLowerCase();
      if (!value) return;
      const actorProfile = approval.actor?.id ? liveVersionOwnerProfilesById?.get(approval.actor.id) : undefined;
      const actorName = approval.actor?.name || actorProfile?.name || "Unknown user";
      let title = "Approval Updated";
      let detail = "updated approval state.";
      let tone: "slate" | "emerald" | "rose" | "amber" = "slate";
      if (value.includes("unapproved")) {
        title = "Unapproved";
        detail = "removed their approval.";
      } else if (value.includes("approved")) {
        title = "Approved";
        detail = "approved this change request.";
        tone = "emerald";
      } else if (value.includes("rejected")) {
        title = "Rejected";
        detail = "rejected this change request.";
        tone = "rose";
      }
      let invalidated = false;
      if (approval.invalidatedAt && value.includes("approved") && !value.includes("unapproved")) {
        invalidated = true;
        tone = "amber";
      }

      items.push({
        id: `approval-${approval.createdAt || index}`,
        title,
        detail,
        timestamp: formatTimestamp(approval.createdAt) || "unknown time",
        tone,
        invalidated,
        actorName,
      });
    });

    if (itemStatus === "published") {
      items.push({
        id: "published",
        title: "Published",
        detail: "This change request was published to live.",
        timestamp:
          formatTimestamp(effectiveChangeRequest.metadata?.publishedAt || effectiveChangeRequest.metadata?.updatedAt) ||
          "unknown time",
        tone: "emerald",
      });
    }

    return items.reverse();
  }, [effectiveChangeRequest, itemStatus, liveVersionOwnerProfilesById]);

  return (
    <Dialog open={!!context} onOpenChange={onOpenChange}>
      <DialogContent className="min-w-[60vw] max-w-5xl max-h-[92vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-base">
            {effectiveChangeRequest?.metadata?.title?.trim() || "Version Node Diff"}
          </DialogTitle>
          {effectiveChangeRequest ? (
            <DialogDescription asChild className="text-[13px] text-slate-600">
              <div className="text-left">
                <p className="m-0 flex flex-wrap items-center gap-x-3 gap-y-1 leading-normal">
                  <span>
                    {reviewPhase.kind !== "none" ? (
                      <>
                        <span className={cn("mr-1", reviewPhase.dotClassName)}>{"\u25cf"}</span>
                        <span className={reviewPhase.labelClassName}>{reviewPhase.label}</span>
                        {" \u00b7 "}
                      </>
                    ) : null}
                    {diffOwner?.name || "Unknown user"} on {formatTimestamp(effectiveChangeRequest.metadata?.createdAt)}
                  </span>
                  <span>Changed Nodes: {diffChangedCount}</span>
                  <span className={cn(diffHasConflicts ? "font-semibold text-red-700" : "text-emerald-700")}>
                    Conflicts: {diffConflictCount}
                  </span>
                </p>
              </div>
            </DialogDescription>
          ) : null}
        </DialogHeader>

        {showReviewActionsSection ? (
          <div
            className={cn(
              "space-y-2 rounded-md p-4 text-left",
              reviewPhase.kind !== "none" ? reviewPhase.sidebarRowActiveClassName : "bg-orange-100",
            )}
          >
            <p className="text-sm font-semibold text-slate-900">Review Actions</p>
            <p className="text-[13px] text-slate-600">
              Active approvals: {diffActiveApprovedCount}/{diffRequiredApprovalsCount}
            </p>
            <div className="flex flex-wrap gap-2">
              {canPublish && onPublishChangeRequest && !changeRequestActionPending ? (
                <Button type="button" onClick={() => onPublishChangeRequest(changeRequestId)}>
                  Publish
                </Button>
              ) : null}
              {canApprove && onApproveChangeRequest && !changeRequestActionPending ? (
                <Button type="button" variant="outline" onClick={() => onApproveChangeRequest(changeRequestId)}>
                  Approve
                </Button>
              ) : null}
              {canUnapprove && onUnapproveChangeRequest && !changeRequestActionPending ? (
                <Button type="button" variant="outline" onClick={() => onUnapproveChangeRequest(changeRequestId)}>
                  Unapprove
                </Button>
              ) : null}
              {canReject && onRejectChangeRequest && !changeRequestActionPending ? (
                <Button type="button" variant="outline" onClick={() => onRejectChangeRequest(changeRequestId)}>
                  Reject
                </Button>
              ) : null}
              {canReopen && onReopenChangeRequest && !changeRequestActionPending ? (
                <Button type="button" variant="outline" onClick={() => onReopenChangeRequest(changeRequestId)}>
                  Reopen
                </Button>
              ) : null}
            </div>
            {diffHasConflicts ? (
              <div
                className={cn(
                  "mt-3 border-t pt-3",
                  reviewPhase.kind === "approved-not-published" ? "border-yellow-200/80" : "border-orange-200/80",
                )}
              >
                <p className="text-[13px] text-slate-600">
                  Conflicts found in this request. Open resolver to merge node changes.
                </p>
                <Button
                  type="button"
                  className="mt-2"
                  variant="secondary"
                  disabled={!canResolveConflicts || !!resolvePending || !onGoToVersioningToResolveConflicts}
                  onClick={() => onGoToVersioningToResolveConflicts?.(changeRequestId)}
                >
                  Resolve Conflicts
                </Button>
              </div>
            ) : null}
          </div>
        ) : null}

        {!diffSummary ? null : (
          <div className="space-y-3">
            {effectiveChangeRequest ? (
              <div className="w-full space-y-3 text-xs text-slate-700">
                <div className="mb-6">
                  <p className="text-sm font-semibold text-slate-900">Summary</p>
                  <div className="mt-2">
                    <VersionNodeDiffAccordion summary={diffSummary} />
                  </div>
                </div>

                <div>
                  <p className="text-sm font-semibold text-slate-900">Activity</p>
                  <ol className="mt-3 space-y-3 text-[13px]">
                    {diffActivityItems.map((item, index) => (
                      <li key={item.id} className="relative flex items-start gap-3">
                        <div className="relative flex w-3 justify-center">
                          {index < diffActivityItems.length - 1 ? (
                            <span className="absolute left-1/2 top-4 h-[calc(100%+1.5rem)] w-px -translate-x-1/2 bg-slate-200" />
                          ) : null}
                          <span
                            className={cn(
                              "mt-1 h-2.5 w-2.5 rounded-full",
                              item.tone === "emerald"
                                ? "bg-emerald-500"
                                : item.tone === "rose"
                                  ? "bg-rose-500"
                                  : item.tone === "amber"
                                    ? "bg-amber-500"
                                    : "bg-slate-400",
                            )}
                          />
                        </div>
                        <div className="min-w-0">
                          <p className="font-medium text-slate-900">
                            {item.title}
                            {item.invalidated ? <span className="ml-1 text-amber-600">(invalidated)</span> : null}
                            <span className="font-normal text-slate-500">· {item.timestamp}</span>
                          </p>
                          <p className="mt-1 flex items-center gap-1.5 text-slate-600">
                            {item.actorName ? (
                              <span className="font-medium text-slate-900">{item.actorName}</span>
                            ) : null}
                            <span>{item.detail}</span>
                          </p>
                        </div>
                      </li>
                    ))}
                  </ol>
                </div>
              </div>
            ) : (
              <VersionNodeDiffAccordion summary={diffSummary} />
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

import { CheckCircle2, ShieldCheck, XCircle } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Textarea } from "@/components/ui/textarea";
import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";

import { OrgUserReference } from "../OrgUserReference";
import type { WorkOrderTimelineApproval, WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";

interface ApprovalTimelineBodyProps {
  event: WorkOrderTimelineEvent;
  approval: WorkOrderTimelineApproval;
  actorDisplay: ReturnType<OrgUserDisplayLookup>;
  resolveUserDisplay: OrgUserDisplayLookup;
  canResolve: boolean;
  isResolving: boolean;
  onResolve?: (input: {
    approvalId: string;
    status: "STATUS_APPROVED" | "STATUS_REJECTED";
    comment?: string;
  }) => Promise<void>;
}

/**
 * Renders both approval events on the activity feed:
 *   - `approvalRequested`: pending → inline card with Approve / Reject buttons.
 *   - `approvalResolved`: plain "X approved/rejected plan" line with the
 *     approver's comment when present.
 *
 * The parent decides which one is rendered based on `event.kind`; this
 * component branches internally to keep call sites simple.
 */
export function ApprovalTimelineBody({
  event,
  approval,
  actorDisplay,
  resolveUserDisplay,
  canResolve,
  isResolving,
  onResolve,
}: ApprovalTimelineBodyProps) {
  if (event.kind === "approvalRequested") {
    return (
      <PendingApprovalCard
        event={event}
        approval={approval}
        actorDisplay={actorDisplay}
        resolveUserDisplay={resolveUserDisplay}
        canResolve={canResolve}
        isResolving={isResolving}
        onResolve={onResolve}
      />
    );
  }

  return <ResolvedApprovalLine event={event} approval={approval} actorDisplay={actorDisplay} />;
}

function PendingApprovalCard({
  event,
  approval,
  actorDisplay,
  resolveUserDisplay,
  canResolve,
  isResolving,
  onResolve,
}: ApprovalTimelineBodyProps) {
  const approverDisplay = approval.approverId ? resolveUserDisplay(approval.approverId) : null;
  const requesterDisplay = actorDisplay;
  const isPending = approval.status === "pending";

  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50/50 p-3 dark:border-amber-500/30 dark:bg-amber-500/5">
      <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
        {requesterDisplay ? (
          <OrgUserReference display={requesterDisplay} size="sm" emphasizeName />
        ) : (
          <span className="font-semibold">Someone</span>
        )}
        <span>requested approval —</span>
        <span className="font-medium">{approval.title}</span>
      </p>

      {approval.message ? (
        <p className="mt-2 whitespace-pre-wrap text-sm text-gray-700 dark:text-gray-300">{approval.message}</p>
      ) : null}

      {approverDisplay ? (
        <p className="mt-2 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
          <OrgUserReference display={approverDisplay} size="sm" />
          <span>· Waiting · {formatTimeAgo(new Date(event.at))}</span>
        </p>
      ) : null}

      {isPending ? (
        <ApprovalResolveForm
          approval={approval}
          canResolve={canResolve}
          isResolving={isResolving}
          onResolve={onResolve}
        />
      ) : null}
    </div>
  );
}

function ResolvedApprovalLine({
  event,
  approval,
  actorDisplay,
}: {
  event: WorkOrderTimelineEvent;
  approval: WorkOrderTimelineApproval;
  actorDisplay: ReturnType<OrgUserDisplayLookup>;
}) {
  const Icon = approval.status === "approved" ? CheckCircle2 : approval.status === "rejected" ? XCircle : ShieldCheck;
  const verb = approval.status === "approved" ? "approved" : approval.status === "rejected" ? "rejected" : "resolved";
  const tone =
    approval.status === "approved"
      ? "text-emerald-600 dark:text-emerald-400"
      : approval.status === "rejected"
        ? "text-red-600 dark:text-red-400"
        : "text-gray-500";

  return (
    <div>
      <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
        {actorDisplay ? (
          <OrgUserReference display={actorDisplay} size="sm" emphasizeName />
        ) : (
          <span className="font-semibold">Someone</span>
        )}
        <Icon className={cn("h-3.5 w-3.5", tone)} aria-hidden />
        <span>{verb}</span>
        <span className="font-medium">{approval.title}</span>
      </p>

      {approval.comment ? (
        <p className="mt-2 rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:border-gray-700/70 dark:bg-gray-900/40 dark:text-gray-300">
          {approval.comment}
        </p>
      ) : null}

      <time className="mt-1 block text-xs text-gray-500 dark:text-gray-400">{formatTimeAgo(new Date(event.at))}</time>
    </div>
  );
}

function ApprovalResolveForm({
  approval,
  canResolve,
  isResolving,
  onResolve,
}: {
  approval: WorkOrderTimelineApproval;
  canResolve: boolean;
  isResolving: boolean;
  onResolve?: (input: {
    approvalId: string;
    status: "STATUS_APPROVED" | "STATUS_REJECTED";
    comment?: string;
  }) => Promise<void>;
}) {
  const [comment, setComment] = useState("");
  const [pendingStatus, setPendingStatus] = useState<"STATUS_APPROVED" | "STATUS_REJECTED" | null>(null);

  const submit = async (status: "STATUS_APPROVED" | "STATUS_REJECTED") => {
    if (!onResolve) return;
    try {
      setPendingStatus(status);
      await onResolve({
        approvalId: approval.id,
        status,
        comment: comment.trim() || undefined,
      });
      setComment("");
    } finally {
      setPendingStatus(null);
    }
  };

  return (
    <div className="mt-3 space-y-2">
      <Textarea
        value={comment}
        onChange={(event) => setComment(event.target.value)}
        placeholder="Leave a note for the requester (optional)"
        rows={2}
        className="text-sm"
        disabled={!canResolve || isResolving}
        data-testid="work-order-approval-comment"
      />

      <div className="flex items-center gap-2">
        <PermissionTooltip allowed={canResolve} message="You don't have permission to resolve approvals.">
          <LoadingButton
            type="button"
            size="sm"
            disabled={!canResolve || isResolving}
            loading={pendingStatus === "STATUS_APPROVED"}
            loadingText="Approving..."
            onClick={() => void submit("STATUS_APPROVED")}
            data-testid="work-order-approval-approve"
          >
            Approve
          </LoadingButton>
        </PermissionTooltip>

        <PermissionTooltip allowed={canResolve} message="You don't have permission to resolve approvals.">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={!canResolve || isResolving}
            onClick={() => void submit("STATUS_REJECTED")}
            data-testid="work-order-approval-reject"
          >
            {pendingStatus === "STATUS_REJECTED" ? "Rejecting…" : "Reject"}
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}

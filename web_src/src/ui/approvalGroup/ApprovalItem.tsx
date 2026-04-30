import * as React from "react";
import { cn } from "@/lib/utils";
import { Check, Circle, MessageCircle, Paperclip, X } from "lucide-react";
import { Button } from "../button";
import { LoadingButton } from "@/components/ui/loading-button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "../hoverCard";
import { Input } from "../input";
import { Item, ItemContent, ItemTitle } from "../item";
import { Label } from "../label";

export interface ApprovalItemProps {
  id: string;
  title: string;
  approved?: boolean;
  rejected?: boolean;
  href?: string;
  className?: string;
  interactive?: boolean;
  onApprove?: (comment?: string) => void | Promise<void>;
  onReject?: (reason: string) => void | Promise<void>;
  approverName?: string;
  approverAvatar?: string;
  approvalComment?: string;
  rejectionReason?: string;
}

export const ApprovalItem: React.FC<ApprovalItemProps> = ({
  title,
  approved = false,
  rejected = false,
  className,
  interactive = false,
  onApprove,
  onReject,
  approverName,
  approverAvatar,
  approvalComment,
  rejectionReason,
}) => {
  const [showRejectionForm, setShowRejectionForm] = React.useState(false);
  const [showApprovalForm, setShowApprovalForm] = React.useState(false);
  const [rejectionReasonInput, setRejectionReasonInput] = React.useState("");
  const [approvalCommentInput, setApprovalCommentInput] = React.useState("");
  const [isApproving, setIsApproving] = React.useState(false);
  const [isRejecting, setIsRejecting] = React.useState(false);

  const content = (
    <>
      <div className="flex items-center justify-center">
        {approved ? (
          <div className="flex size-5 items-center justify-center rounded-full bg-emerald-500">
            <Check className="size-3 text-white" />
          </div>
        ) : rejected ? (
          <div className="flex size-5 items-center justify-center rounded-full bg-red-500">
            <X className="size-3 text-white" />
          </div>
        ) : (
          <Circle className="size-5 text-muted-foreground" strokeDasharray="4 4" />
        )}
      </div>
      <ItemContent>
        <ItemTitle className="text-base font-normal flex items-center gap-2">
          {title}
          {approvalComment && (
            <HoverCard openDelay={150} closeDelay={150}>
              <HoverCardTrigger asChild>
                <span className="flex items-center gap-1 text-muted-foreground cursor-pointer">
                  <Paperclip className="size-4" />
                  <span className="text-sm">{approvalComment}</span>
                </span>
              </HoverCardTrigger>
              <HoverCardContent side="top" className="w-64 space-y-3 text-xs">
                <p className="text-sm font-medium text-neutral-900">Artifacts</p>
                <div className="space-y-3">
                  <p className="font-medium text-neutral-900">Comment</p>
                  <p className="text-muted-foreground">{approvalComment}</p>
                </div>
              </HoverCardContent>
            </HoverCard>
          )}
          {rejected && rejectionReason && (
            <HoverCard openDelay={150} closeDelay={150}>
              <HoverCardTrigger asChild>
                <span className="flex items-center gap-1 text-muted-foreground cursor-pointer">
                  <MessageCircle className="size-4" />
                </span>
              </HoverCardTrigger>
              <HoverCardContent side="top" className="w-64 text-xs">
                <p className="text-sm font-medium text-neutral-900 mb-2">Rejection Reason</p>
                <p className="text-muted-foreground">{rejectionReason}</p>
              </HoverCardContent>
            </HoverCard>
          )}
        </ItemTitle>
      </ItemContent>
      {interactive && !showRejectionForm && !showApprovalForm ? (
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            className="h-7 py-1 px-2"
            disabled={isApproving}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setShowRejectionForm(true);
            }}
          >
            Reject
          </Button>
          <LoadingButton
            variant="default"
            className="h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
            loading={isApproving}
            loadingText="Approving..."
            onClick={async (e) => {
              e.preventDefault();
              e.stopPropagation();
              setShowApprovalForm(true);
            }}
          >
            Approve
          </LoadingButton>
        </div>
      ) : (interactive && showRejectionForm) || (interactive && showApprovalForm) ? (
        <span></span>
      ) : (
        <div className="flex items-center gap-2">
          {(approved || rejected) && approverAvatar ? (
            <>
              <img
                src={approverAvatar}
                alt={approverName || (approved ? "Approver" : "Rejector")}
                className="size-8 rounded-full object-cover"
              />
              <span className="text-base font-normal text-muted-foreground">
                {approverName || (approved ? "Approved" : "Rejected")}
              </span>
            </>
          ) : (
            <span
              className={cn(
                "text-base font-normal",
                approved ? "text-green-700" : rejected ? "text-red-700" : "text-muted-foreground",
              )}
            >
              {approved ? "Approved" : rejected ? "Rejected" : "Pending"}
            </span>
          )}
        </div>
      )}
    </>
  );

  if (interactive) {
    return (
      <>
        <Item variant="outline" size="sm" className={cn("w-full border-0  border-border px-2 py-1.5", className)}>
          {content}
        </Item>
        {showRejectionForm && (
          <div
            className="w-full border bg-gray-50 px-3 py-2 my-2 rounded-lg text-left"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex flex-col gap-4">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <Label htmlFor="rejection-comment" className="text-sm font-semibold text-gray-800">
                    Comment
                  </Label>
                </div>
              </div>
              <Input
                id="rejection-comment"
                placeholder="Reason for rejection…"
                value={rejectionReasonInput}
                onChange={(e) => setRejectionReasonInput(e.target.value)}
                className="w-full outline-none focus-visible:ring-0 focus-visible:ring-offset-0 "
              />
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  size="default"
                  className="h-7 py-1 px-2"
                  disabled={isRejecting}
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowRejectionForm(false);
                    setRejectionReasonInput("");
                  }}
                >
                  Cancel
                </Button>
                <LoadingButton
                  variant="default"
                  size="default"
                  className="h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
                  loading={isRejecting}
                  loadingText="Rejecting..."
                  onClick={async (e) => {
                    e.stopPropagation();
                    setIsRejecting(true);
                    try {
                      await onReject?.(rejectionReasonInput);
                    } finally {
                      setIsRejecting(false);
                    }
                    setShowRejectionForm(false);
                    setRejectionReasonInput("");
                  }}
                  disabled={!rejectionReasonInput.trim()}
                >
                  Confirm Rejection
                </LoadingButton>
              </div>
            </div>
          </div>
        )}
        {showApprovalForm && (
          <div
            className="w-full border my-2 bg-gray-50 px-3 py-2 rounded-lg text-left"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <Label htmlFor="approval-comment" className="text-sm font-semibold text-neutral-900">
                      Comment
                    </Label>
                  </div>
                </div>
                <Input
                  id="approval-comment"
                  placeholder="Enter comment..."
                  value={approvalCommentInput}
                  onChange={(e) => setApprovalCommentInput(e.target.value)}
                  className="w-full outline-none  focus-visible:ring-0 focus-visible:ring-offset-0"
                />
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  className="h-7 py-1 px-2"
                  disabled={isApproving}
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowApprovalForm(false);
                    setApprovalCommentInput("");
                  }}
                >
                  Cancel
                </Button>
                <LoadingButton
                  variant="default"
                  className="h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
                  loading={isApproving}
                  loadingText="Approving..."
                  onClick={async (e) => {
                    e.stopPropagation();
                    setIsApproving(true);
                    try {
                      await onApprove?.(approvalCommentInput);
                    } finally {
                      setIsApproving(false);
                    }
                    setShowApprovalForm(false);
                    setApprovalCommentInput("");
                  }}
                >
                  Confirm Approval
                </LoadingButton>
              </div>
            </div>
          </div>
        )}
      </>
    );
  }

  return (
    <Item variant="outline" size="sm" className={cn("w-full border-0 border-t border-border px-2 py-1.5", className)}>
      {content}
    </Item>
  );
};

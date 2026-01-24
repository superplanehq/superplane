import * as React from "react";

import { cn } from "@/lib/utils";

import { Check, Circle, MessageCircle, Paperclip, X } from "lucide-react";
import { Button } from "../button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "../hoverCard";
import { Input } from "../input";
import { Item, ItemContent, ItemTitle } from "../item";
import { Label } from "../label";

export interface ArtifactField {
  label: string;
  optional?: boolean;
}

export interface ApprovalItemProps {
  id: string;
  title: string;
  approved?: boolean;
  rejected?: boolean;
  href?: string;
  className?: string;
  interactive?: boolean;
  onApprove?: (artifacts?: Record<string, string>) => void;
  onReject?: (comment?: string) => void;
  approverName?: string;
  approverAvatar?: string;
  requireArtifacts?: ArtifactField[];
  artifactCount?: number;
  artifacts?: Record<string, string>;
  rejectionComment?: string;
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
  requireArtifacts = [],
  artifactCount,
  artifacts: providedArtifacts,
  rejectionComment,
}) => {
  const [showRejectionForm, setShowRejectionForm] = React.useState(false);
  const [showApprovalForm, setShowApprovalForm] = React.useState(false);
  const [rejectionCommentInput, setRejectionCommentInput] = React.useState("");
  const [artifacts, setArtifacts] = React.useState<Record<string, string>>({});

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
          {artifactCount !== undefined && artifactCount > 0 && providedArtifacts && (
            <HoverCard openDelay={150} closeDelay={150}>
              <HoverCardTrigger asChild>
                <span className="flex items-center gap-1 text-muted-foreground cursor-pointer">
                  <Paperclip className="size-4" />
                  <span className="text-sm">{artifactCount}</span>
                </span>
              </HoverCardTrigger>
              <HoverCardContent side="top" className="w-64 space-y-3 text-xs">
                <p className="text-sm font-medium text-neutral-900 dark:text-gray-200">Artifacts</p>
                <div className="space-y-3">
                  {Object.entries(providedArtifacts).map(([name, value]) => (
                    <div key={name} className="space-y-1">
                      <p className="font-medium text-neutral-900 dark:text-gray-200">{name}</p>
                      <a
                        href={value}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-blue-600 hover:text-blue-800 underline block break-all text-xs"
                      >
                        {value}
                      </a>
                    </div>
                  ))}
                </div>
              </HoverCardContent>
            </HoverCard>
          )}
          {rejected && rejectionComment && (
            <HoverCard openDelay={150} closeDelay={150}>
              <HoverCardTrigger asChild>
                <span className="flex items-center gap-1 text-muted-foreground cursor-pointer">
                  <MessageCircle className="size-4" />
                </span>
              </HoverCardTrigger>
              <HoverCardContent side="top" className="w-64 text-xs">
                <p className="text-sm font-medium text-neutral-900 dark:text-gray-200 mb-2">Rejection Comment</p>
                <p className="text-muted-foreground">{rejectionComment}</p>
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
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              setShowRejectionForm(true);
            }}
          >
            Reject
          </Button>
          <Button
            variant="default"
            className="h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              if (requireArtifacts.length > 0) {
                setShowApprovalForm(true);
              } else {
                onApprove?.();
              }
            }}
          >
            Approve
          </Button>
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
            className="w-full border dark:border-gray-600 bg-gray-50 dark:bg-gray-800 px-3 py-2 my-2 rounded-lg text-left"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex flex-col gap-4">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <Label htmlFor="rejection-comment" className="text-sm font-semibold text-gray-800 dark:text-gray-200">
                    Comment
                  </Label>
                </div>
              </div>
              <Input
                id="rejection-comment"
                placeholder="Reason for rejection…"
                value={rejectionCommentInput}
                onChange={(e) => setRejectionCommentInput(e.target.value)}
                className="w-full outline-none focus-visible:ring-0 focus-visible:ring-offset-0 "
              />
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  size="default"
                  className="h-7 py-1 px-2"
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowRejectionForm(false);
                    setRejectionCommentInput("");
                  }}
                >
                  Cancel
                </Button>
                <Button
                  variant="default"
                  size="default"
                  className="h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
                  onClick={(e) => {
                    e.stopPropagation();
                    onReject?.(rejectionCommentInput);
                    setShowRejectionForm(false);
                    setRejectionCommentInput("");
                  }}
                  disabled={!rejectionCommentInput.trim()}
                >
                  Confirm Rejection
                </Button>
              </div>
            </div>
          </div>
        )}
        {showApprovalForm && (
          <div
            className="w-full border dark:border-gray-600 my-2 bg-gray-50 dark:bg-gray-800 px-3 py-2 rounded-lg text-left"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex flex-col gap-4">
              {requireArtifacts.map((artifact, index) => (
                <div key={index} className="flex flex-col gap-2">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <Label htmlFor={`artifact-${index}`} className="text-sm font-semibold text-neutral-900 dark:text-gray-200">
                        {artifact.label}
                      </Label>
                      <span className="ml-2 text-sm text-muted-foreground">{artifact.optional ? "Optional" : ""}</span>
                    </div>
                  </div>
                  <Input
                    id={`artifact-${index}`}
                    placeholder={"Enter " + artifact.label + "..."}
                    value={artifacts[artifact.label] || ""}
                    onChange={(e) =>
                      setArtifacts((prev) => ({
                        ...prev,
                        [artifact.label]: e.target.value,
                      }))
                    }
                    className="w-full outline-none  focus-visible:ring-0 focus-visible:ring-offset-0"
                  />
                </div>
              ))}
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  className="h-7 py-1 px-2"
                  onClick={(e) => {
                    e.stopPropagation();
                    setShowApprovalForm(false);
                    setArtifacts({});
                  }}
                >
                  Cancel
                </Button>
                <Button
                  variant="default"
                  className="h-7 py-1 px-2 bg-black text-white hover:bg-black/80"
                  onClick={(e) => {
                    e.stopPropagation();
                    onApprove?.(artifacts);
                    setShowApprovalForm(false);
                    setArtifacts({});
                  }}
                  disabled={requireArtifacts
                    .filter((artifact) => !artifact.optional)
                    .some((artifact) => !artifacts[artifact.label]?.trim())}
                >
                  Confirm Approval
                </Button>
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

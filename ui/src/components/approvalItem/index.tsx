import * as React from "react"

import { cn } from "@/lib/utils"

import {
  Item,
  ItemContent,
  ItemTitle,
} from "../item"
import { Button } from "../button"
import { Input } from "../input"
import { Label } from "../label"
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "../hoverCard"
import { Check, Circle, MessageCircle, Paperclip, X } from "lucide-react"

export interface ArtifactField {
  label: string
  optional?: boolean
}

export interface ApprovalItemProps {
  title: string
  approved?: boolean
  rejected?: boolean
  href?: string
  className?: string
  interactive?: boolean
  onApprove?: (artifacts?: Record<string, string>) => void
  onReject?: (comment?: string) => void
  approverName?: string
  approverAvatar?: string
  requireArtifacts?: ArtifactField[]
  artifactCount?: number
  artifacts?: Record<string, string>
  rejectionComment?: string
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
  const [showRejectionForm, setShowRejectionForm] = React.useState(false)
  const [showApprovalForm, setShowApprovalForm] = React.useState(false)
  const [rejectionCommentInput, setRejectionCommentInput] = React.useState("")
  const [artifacts, setArtifacts] = React.useState<Record<string, string>>({})

  const content = (
    <>
      <div className="flex items-center justify-center">
        {approved ? (
          <div className="flex size-6 items-center justify-center rounded-full bg-emerald-500">
            <Check className="size-4 text-white" />
          </div>
        ) : rejected ? (
          <div className="flex size-6 items-center justify-center rounded-full bg-red-500">
            <X className="size-4 text-white" />
          </div>
        ) : (
          <Circle className="size-6 text-muted-foreground" strokeDasharray="4 4" />
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
                <p className="text-sm font-medium text-neutral-900">
                  Artifacts
                </p>
                <div className="space-y-3">
                  {Object.entries(providedArtifacts).map(([name, value]) => (
                    <div key={name} className="space-y-1">
                      <p className="font-medium text-neutral-900">{name}</p>
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
                <p className="text-sm font-medium text-neutral-900 mb-2">
                  Rejection Comment
                </p>
                <p className="text-muted-foreground">
                  {rejectionComment}
                </p>
              </HoverCardContent>
            </HoverCard>
          )}
        </ItemTitle>
      </ItemContent>
      {interactive && !showRejectionForm && !showApprovalForm ? (
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={(e) => {
              e.preventDefault()
              setShowRejectionForm(true)
            }}
          >
            Reject
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={(e) => {
              e.preventDefault()
              if (requireArtifacts.length > 0) {
                setShowApprovalForm(true)
              } else {
                onApprove?.()
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
            <span className={cn(
              "text-base font-normal",
              approved ? "text-emerald-500" : rejected ? "text-red-500" : "text-muted-foreground"
            )}>
              {approved ? "Approved" : rejected ? "Rejected" : "Pending"}
            </span>
          )}
        </div>
      )}
    </>
  )

  if (interactive) {
    return (
      <>
        <Item variant="outline" size="sm" className={cn("w-full border-0 border-t border-border px-2 py-1.5", className)}>
          {content}
        </Item>
        {showRejectionForm && (
          <div className="w-full border-0 border-t border-border bg-gray-50 p-6 rounded-lg">
            <div className="flex flex-col gap-4">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <Label htmlFor="rejection-comment" className="text-sm font-semibold text-neutral-900">
                    Comment
                  </Label>
                  <span className="ml-2 text-sm text-muted-foreground">Required</span>
                </div>
              </div>
              <Input
                id="rejection-comment"
                placeholder="Enter the reason for rejection..."
                value={rejectionCommentInput}
                onChange={(e) => setRejectionCommentInput(e.target.value)}
                className="w-full"
              />
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  size="default"
                  onClick={() => {
                    setShowRejectionForm(false)
                    setRejectionCommentInput("")
                  }}
                >
                  Cancel
                </Button>
                <Button
                  variant="default"
                  size="default"
                  onClick={() => {
                    onReject?.(rejectionCommentInput)
                    setShowRejectionForm(false)
                    setRejectionCommentInput("")
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
          <div className="w-full border-0 border-t border-border bg-gray-50 p-6 rounded-lg">
            <div className="flex flex-col gap-4">
              {requireArtifacts.map((artifact, index) => (
                <div key={index} className="flex flex-col gap-2">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <Label htmlFor={`artifact-${index}`} className="text-sm font-semibold text-neutral-900">
                        {artifact.label}
                      </Label>
                      <span className="ml-2 text-sm text-muted-foreground">
                        {artifact.optional ? "Optional" : "Required"}
                      </span>
                    </div>
                  </div>
                  <Input
                    id={`artifact-${index}`}
                    placeholder="Please provide link to the artifact"
                    value={artifacts[artifact.label] || ""}
                    onChange={(e) => setArtifacts(prev => ({ ...prev, [artifact.label]: e.target.value }))}
                    className="w-full"
                  />
                </div>
              ))}
              <div className="flex items-center justify-end gap-2">
                <Button
                  variant="outline"
                  size="default"
                  onClick={() => {
                    setShowApprovalForm(false)
                    setArtifacts({})
                  }}
                >
                  Cancel
                </Button>
                <Button
                  variant="default"
                  size="default"
                  onClick={() => {
                    onApprove?.(artifacts)
                    setShowApprovalForm(false)
                    setArtifacts({})
                  }}
                  disabled={requireArtifacts.filter(artifact => !artifact.optional).some(artifact => !artifacts[artifact.label]?.trim())}
                >
                  Confirm Approval
                </Button>
              </div>
            </div>
          </div>
        )}
      </>
    )
  }

  return (
    <Item variant="outline" size="sm" className={cn("w-full border-0 border-t border-border px-2 py-1.5", className)}>
      {content}
    </Item>
  )
}

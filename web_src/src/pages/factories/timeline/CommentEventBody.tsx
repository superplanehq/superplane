import { Link } from "@/components/Link/link";
import { MarkdownContent } from "@/pages/app/Markdown";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";
import { getWorkOrderRunHref } from "../lib/workOrderExecutions";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import {
  timelineActorClassName as inlineActorClassName,
  timelineLinkClassName as inlineLinkClassName,
  timelineParagraphClassName as inlineParagraphClassName,
  timelineTimeClassName as inlineTimeClassName,
} from "./timelineStyles";
import { CommentReactions } from "./CommentReactions";
import { TimelineAutomationActor } from "./TimelineAutomationActor";

interface CommentEventBodyProps {
  event: WorkOrderTimelineEvent;
  actorDisplay: OrgUserDisplay | null;
  timeLabel: string;
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  canReactToComments: boolean;
  onAddCommentReaction?: (commentId: string, emoji: string) => void;
  onRemoveCommentReaction?: (commentId: string, emoji: string) => void;
}

export function CommentEventBody({
  event,
  actorDisplay,
  timeLabel,
  organizationId,
  factoryKey,
  orderNumber,
  canReactToComments,
  onAddCommentReaction,
  onRemoveCommentReaction,
}: CommentEventBodyProps) {
  const comment = event.comment;
  if (!comment) return null;
  const isAutomation = (comment.authorKind ?? "").toLowerCase() === "automation";
  const runHref = getWorkOrderRunHref(organizationId, factoryKey, event.sourceAppId, event.sourceRunId, {
    orderNumber,
  });

  return (
    <>
      <p className={inlineParagraphClassName}>
        {isAutomation && (event.actorAutomation || comment.automation) ? (
          <TimelineAutomationActor actor={event.actorAutomation ?? comment.automation!} fallbackLabel="Automation" />
        ) : actorDisplay ? (
          <span className={inlineActorClassName}>{actorDisplay.name}</span>
        ) : (
          <span className={inlineActorClassName}>Someone</span>
        )}{" "}
        commented
        {isAutomation && runHref ? (
          <>
            {" "}
            via{" "}
            <Link href={runHref} className={inlineLinkClassName}>
              run
            </Link>
          </>
        ) : null}
        <span className={inlineTimeClassName}>
          {" · "}
          {timeLabel}
        </span>
      </p>
      <div className="mt-1" data-testid="work-order-timeline-comment-body">
        <MarkdownContent content={comment.body} variant="workspace" />
      </div>
      <CommentReactions
        reactions={comment.reactions}
        canReact={canReactToComments}
        onAddReaction={(emoji) => onAddCommentReaction?.(comment.id, emoji)}
        onRemoveReaction={(emoji) => onRemoveCommentReaction?.(comment.id, emoji)}
      />
    </>
  );
}

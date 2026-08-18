import { Link } from "@/components/Link/link";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";
import { MarkdownContent } from "@/pages/app/Markdown";

import { getWorkOrderRunHref } from "../lib/workOrderExecutions";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { TimelineAutomationActor } from "./TimelineAutomationActor";
import {
  timelineActorClassName,
  timelineLinkClassName,
  timelineParagraphClassName,
  timelineTimeClassName,
} from "./timelineStyles";

export function CommentEventBody({
  event,
  actorDisplay,
  timeLabel,
  organizationId,
  factoryKey,
  orderNumber,
}: {
  event: WorkOrderTimelineEvent;
  actorDisplay: OrgUserDisplay | null;
  timeLabel: string;
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
}) {
  const comment = event.comment;
  if (!comment) return null;
  const isAutomation = (comment.authorKind ?? "").toLowerCase() === "automation";
  const runHref = getWorkOrderRunHref(organizationId, factoryKey, event.sourceAppId, event.sourceRunId, {
    orderNumber,
  });

  return (
    <>
      <p className={timelineParagraphClassName}>
        {isAutomation && (event.actorAutomation || comment.automation) ? (
          <TimelineAutomationActor actor={event.actorAutomation ?? comment.automation!} fallbackLabel="Automation" />
        ) : actorDisplay ? (
          <span className={timelineActorClassName}>{actorDisplay.name}</span>
        ) : (
          <span className={timelineActorClassName}>Someone</span>
        )}{" "}
        commented
        {isAutomation && runHref ? (
          <>
            {" "}
            via{" "}
            <Link href={runHref} className={timelineLinkClassName}>
              run
            </Link>
          </>
        ) : null}
        <span className={timelineTimeClassName}>
          {" · "}
          {timeLabel}
        </span>
      </p>
      <div className="mt-1" data-testid="work-order-timeline-comment-body">
        <MarkdownContent content={comment.body} variant="workspace" />
      </div>
    </>
  );
}

import { Link } from "@/components/Link/link";

import { getWorkOrderRunHref } from "../lib/workOrderExecutions";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { BroadcastActivityBlock } from "./BroadcastActivityBlock";
import { TimelineAutomationActor } from "./TimelineAutomationActor";
import { timelineActorClassName, timelineLinkClassName, timelineParagraphClassName, timelineTimeClassName } from "./timelineStyles";

export function BroadcastEventBody({
  event,
  timeLabel,
  organizationId,
  factoryKey,
  orderNumber,
}: {
  event: WorkOrderTimelineEvent;
  timeLabel: string;
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
}) {
  const broadcast = event.broadcast;
  if (!broadcast) {
    return null;
  }
  const runHref = getWorkOrderRunHref(organizationId, factoryKey, event.sourceAppId, event.sourceRunId, {
    orderNumber,
  });

  return (
    <div className="min-w-0">
      <p className={timelineParagraphClassName}>
        {event.actorAutomation ? (
          <TimelineAutomationActor actor={event.actorAutomation} fallbackLabel="Automation" />
        ) : (
          <span className={timelineActorClassName}>Automation</span>
        )}{" "}
        posted
        {runHref ? (
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
      <div className="mt-1">
        <BroadcastActivityBlock title={broadcast.title} body={broadcast.body} url={broadcast.url} />
      </div>
    </div>
  );
}

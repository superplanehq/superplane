import { Link } from "@/components/Link/link";

import { getWorkOrderRunHref } from "../lib/workOrderExecutions";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { BroadcastContent } from "./BroadcastContent";
import { TimelineAutomationActor } from "./TimelineAutomationActor";
import { timelineLinkClassName, timelineParagraphClassName, timelineTimeClassName } from "./timelineStyles";

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

  const actor = event.actorAutomation;
  const runHref = getWorkOrderRunHref(organizationId, factoryKey, event.sourceAppId, event.sourceRunId, {
    orderNumber,
  });

  return (
    <>
      <p className={timelineParagraphClassName}>
        {actor ? <TimelineAutomationActor actor={actor} fallbackLabel="Automation" /> : <span>Automation</span>} posted
        an update
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
      <BroadcastContent broadcast={broadcast} />
    </>
  );
}

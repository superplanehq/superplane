import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";

import { booleanCheckVerdict, formatCheckScore } from "../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../lib/workOrderExecutions";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { TimelineAutomationActor } from "./TimelineAutomationActor";
import {
  timelineActorClassName,
  timelineLinkClassName,
  timelineParagraphClassName,
  timelineTimeClassName,
} from "./timelineStyles";

/**
 * A check score reported by an automation: "PR Risk Review re-scored
 * Risk review: 82 → 65/100 via run". Re-scores keep the previous value in
 * the sentence so the timeline reads as a trend, not a bare number.
 */
export function CheckReportedEventBody({
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
  const check = event.check;
  if (!check) return null;

  const runHref = getWorkOrderRunHref(organizationId, factoryKey, event.sourceAppId, event.sourceRunId, {
    orderNumber,
  });
  const { value: scoreLabel, scale } = formatCheckScore(check);
  const isBoolean = check.format === "boolean";
  const previousScoreLabel =
    isBoolean && check.previousScore !== undefined ? booleanCheckVerdict(check.previousScore) : check.previousScore;
  const isRescore = check.previousScore !== undefined && check.previousScore !== check.score;

  return (
    <p className={timelineParagraphClassName}>
      {event.actorAutomation ? (
        <TimelineAutomationActor actor={event.actorAutomation} fallbackLabel="Automation" />
      ) : (
        <span className={timelineActorClassName}>Automation</span>
      )}{" "}
      {isRescore ? "re-scored" : "reported"} <span className={timelineActorClassName}>{check.name}</span>:{" "}
      {isRescore ? (
        <>
          <span className="tabular-nums text-muted-foreground">{previousScoreLabel}</span>
          <span className="text-muted-foreground"> → </span>
        </>
      ) : null}
      <span className={cn(timelineActorClassName, "tabular-nums")}>
        {scoreLabel}
        {scale}
      </span>
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
  );
}

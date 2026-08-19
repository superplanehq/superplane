import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";

import { getWorkOrderRunHref } from "../lib/workOrderExecutions";
import type {
  WorkOrderTimelineBooleanCheck,
  WorkOrderTimelineEvent,
  WorkOrderTimelineScoreCheck,
} from "../lib/workOrderTimelineEvents";
import { TimelineAutomationActor } from "./TimelineAutomationActor";
import {
  timelineActorClassName,
  timelineLinkClassName,
  timelineParagraphClassName,
  timelineTimeClassName,
} from "./timelineStyles";

/**
 * A check reported by an automation: "PR Risk Review re-scored Risk
 * review: 82 → 65/100 via run" for a numeric score, or "Security Scan
 * flipped Security scan: Pass → Fail via run" for a boolean check.
 * Re-scores/flips keep the previous value in the sentence so the timeline
 * reads as a trend, not a bare value.
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

  return (
    <p className={timelineParagraphClassName}>
      {event.actorAutomation ? (
        <TimelineAutomationActor actor={event.actorAutomation} fallbackLabel="Automation" />
      ) : (
        <span className={timelineActorClassName}>Automation</span>
      )}{" "}
      {check.type === "boolean" ? <BooleanCheckPhrase check={check} /> : <ScoreCheckPhrase check={check} />}
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

/** "re-scored Risk review: 82 → 65/100" (or "reported … 65/100" for a first report). */
function ScoreCheckPhrase({ check }: { check: WorkOrderTimelineScoreCheck }) {
  const scale = check.format === "percent" ? "%" : `/${check.maxScore}`;
  const isRescore = check.previousScore !== undefined && check.previousScore !== check.score;

  return (
    <>
      {isRescore ? "re-scored" : "reported"} <span className={timelineActorClassName}>{check.name}</span>:{" "}
      {isRescore ? (
        <>
          <span className="tabular-nums text-muted-foreground">{check.previousScore}</span>
          <span className="text-muted-foreground"> → </span>
        </>
      ) : null}
      <span className={cn(timelineActorClassName, "tabular-nums")}>
        {check.score}
        {scale}
      </span>
    </>
  );
}

/** "reported CI: Pass" (or "flipped Security scan: Pass → Fail" when the verdict changed). */
function BooleanCheckPhrase({ check }: { check: WorkOrderTimelineBooleanCheck }) {
  const isFlip = check.previousPassed !== undefined && check.previousPassed !== check.passed;

  return (
    <>
      {isFlip ? "flipped" : "reported"} <span className={timelineActorClassName}>{check.name}</span>:{" "}
      {isFlip ? (
        <>
          <span className="text-muted-foreground">{check.previousPassed ? "Pass" : "Fail"}</span>
          <span className="text-muted-foreground"> → </span>
        </>
      ) : null}
      <span className={timelineActorClassName}>{check.passed ? "Pass" : "Fail"}</span>
    </>
  );
}

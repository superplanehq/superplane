import type { FactoriesFactoryPullRequest } from "@/api-client";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

import { overlayLivePullRequest } from "../lib/workOrderPullRequest";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { WorkOrderPullRequestInline } from "../WorkOrderPullRequestInline";
import { TimelineAutomationActor } from "./TimelineAutomationActor";
import { timelineActorClassName, timelineParagraphClassName, timelineTimeClassName } from "./timelineStyles";

export function PullRequestEventBody({
  event,
  actorDisplay,
  timeLabel,
  latestPullRequestById,
}: {
  event: WorkOrderTimelineEvent;
  actorDisplay: OrgUserDisplay | null;
  timeLabel: string;
  latestPullRequestById: Map<string, FactoriesFactoryPullRequest>;
}) {
  const snapshot = event.pullRequest;
  if (!snapshot) {
    return null;
  }

  const pullRequest = overlayLivePullRequest(snapshot, latestPullRequestById);
  const verb = event.kind === "pullRequestUpdated" ? "updated" : "added";

  return (
    <p className={timelineParagraphClassName}>
      {actorDisplay ? (
        <span className={timelineActorClassName}>{actorDisplay.name}</span>
      ) : event.actorAutomation ? (
        <TimelineAutomationActor actor={event.actorAutomation} />
      ) : (
        <span className={timelineActorClassName}>Someone</span>
      )}{" "}
      {verb} <WorkOrderPullRequestInline pullRequest={pullRequest} className="align-baseline" />
      <span className={timelineTimeClassName}>
        {" · "}
        {timeLabel}
      </span>
    </p>
  );
}

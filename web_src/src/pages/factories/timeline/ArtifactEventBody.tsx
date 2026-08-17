import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

import { overlayLiveArtifactData, type WorkOrderArtifactLike } from "../lib/workOrderArtifact";
import { WorkOrderArtifactInline } from "../WorkOrderArtifactInline";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { TimelineAutomationActor } from "./TimelineAutomationActor";
import { timelineActorClassName, timelineParagraphClassName, timelineTimeClassName } from "./timelineStyles";

export function ArtifactEventBody({
  event,
  actorDisplay,
  timeLabel,
  latestArtifactDataById,
  relatedArtifacts,
}: {
  event: WorkOrderTimelineEvent;
  actorDisplay: OrgUserDisplay | null;
  timeLabel: string;
  latestArtifactDataById: Map<string, Record<string, unknown>>;
  relatedArtifacts: readonly WorkOrderArtifactLike[];
}) {
  const artifact = event.artifact;
  if (!artifact) return null;

  const displayArtifact = overlayLiveArtifactData(artifact, latestArtifactDataById);

  return (
    <p className={timelineParagraphClassName}>
      {actorDisplay ? (
        <span className={timelineActorClassName}>{actorDisplay.name}</span>
      ) : event.actorAutomation ? (
        <TimelineAutomationActor actor={event.actorAutomation} />
      ) : (
        <span className={timelineActorClassName}>Someone</span>
      )}{" "}
      attached{" "}
      <WorkOrderArtifactInline
        artifact={displayArtifact}
        relatedArtifacts={relatedArtifacts}
        className="align-baseline"
      />
      <span className={timelineTimeClassName}>
        {" · "}
        {timeLabel}
      </span>
    </p>
  );
}

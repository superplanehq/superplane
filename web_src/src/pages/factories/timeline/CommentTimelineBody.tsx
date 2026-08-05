import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";

import { OrgUserReference } from "../OrgUserReference";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { resolveCommentAuthorLabel } from "./authorLabels";

export function CommentTimelineBody({
  event,
  actorDisplay,
}: {
  event: WorkOrderTimelineEvent;
  actorDisplay: ReturnType<OrgUserDisplayLookup>;
}) {
  const comment = event.comment;
  if (!comment) {
    return null;
  }

  const kind = (comment.authorKind ?? "").toLowerCase();
  const isAutomation = kind === "automation";
  const isSystem = kind === "system";
  const showAutomationBadge = isAutomation || isSystem;
  //
  // For automation / system comments we deliberately ignore actorDisplay
  // and render the automation identity instead — the OrgUserReference
  // avatar would misrepresent an automated note as a human comment.
  //
  const authorLabel = resolveCommentAuthorLabel({
    isAutomation,
    isSystem,
    automation: comment.automation,
    actorName: actorDisplay?.name,
  });
  const shouldRenderActor = actorDisplay && !showAutomationBadge;

  return (
    <div>
      <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
        {shouldRenderActor ? (
          <OrgUserReference display={actorDisplay} size="sm" emphasizeName />
        ) : (
          <span className="font-semibold">{authorLabel}</span>
        )}
        <span>commented</span>
        {showAutomationBadge ? (
          <span className="ml-1 inline-flex items-center gap-1 rounded border border-violet-200 bg-violet-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200">
            {isSystem ? "System" : "Automation"}
          </span>
        ) : null}
      </p>
      <div className="mt-2 rounded-md border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-800 dark:border-gray-700/70 dark:bg-gray-900/40 dark:text-gray-200">
        <p className="whitespace-pre-wrap leading-relaxed">{comment.body}</p>
      </div>
    </div>
  );
}

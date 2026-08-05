import { ExternalLink, GitPullRequest } from "lucide-react";

import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { OrgUserReference } from "../OrgUserReference";
import { extractArtifactMarkdownBody, formatPrArtifactLabel } from "../lib/workOrderArtifact";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { formatArtifactKindLong } from "./authorLabels";
import { TimelineAutomationActor } from "./TimelineAutomationActor";

export function ArtifactTimelineBody({
  event,
  actorDisplay,
}: {
  event: WorkOrderTimelineEvent;
  actorDisplay: ReturnType<OrgUserDisplayLookup>;
}) {
  const artifact = event.artifact;
  if (!artifact) {
    return null;
  }

  const isPr = artifact.type === "pr";
  const isMarkdown = artifact.type === "markdown";
  const safeUrl = safeExternalUrl(artifact.url);
  const prLabel = isPr ? formatPrArtifactLabel(artifact.data) : undefined;
  const label = prLabel || artifact.title?.trim() || safeUrl || (isPr ? "Pull request" : "Note");
  const markdownBody = isMarkdown ? extractArtifactMarkdownBody(artifact.data) : undefined;
  const automationActor = event.actorAutomation;

  return (
    <div>
      <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
        {actorDisplay ? (
          <OrgUserReference display={actorDisplay} size="sm" emphasizeName />
        ) : automationActor ? (
          <TimelineAutomationActor actor={automationActor} />
        ) : (
          <span className="font-semibold">Someone</span>
        )}
        <span>attached a</span>
        <span className="font-medium">{formatArtifactKindLong(artifact.type)}</span>
        {automationActor && !actorDisplay ? (
          <span className="ml-1 inline-flex items-center gap-1 rounded border border-violet-200 bg-violet-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200">
            Automation
          </span>
        ) : null}
      </p>
      <div className="mt-2 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-800 dark:border-gray-700/70 dark:bg-gray-900/40 dark:text-gray-200">
        {safeUrl ? (
          <a
            href={safeUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex max-w-full items-center gap-2 font-medium text-violet-700 hover:underline dark:text-violet-300"
          >
            {isPr ? (
              <GitPullRequest className="h-4 w-4" aria-hidden />
            ) : (
              <ExternalLink className="h-4 w-4" aria-hidden />
            )}
            <span className="truncate">{label}</span>
          </a>
        ) : (
          <p className="font-medium">{label}</p>
        )}
        {markdownBody ? (
          <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-gray-700 dark:text-gray-300">
            {markdownBody}
          </p>
        ) : null}
      </div>
    </div>
  );
}


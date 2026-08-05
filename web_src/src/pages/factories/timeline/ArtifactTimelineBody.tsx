import { ExternalLink, GitPullRequest } from "lucide-react";

import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { OrgUserReference } from "../OrgUserReference";
import type { WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { formatArtifactKindLong } from "./authorLabels";

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
  const label = artifact.title?.trim() || safeUrl || (isPr ? "Pull request" : "Note");

  return (
    <div>
      <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
        {actorDisplay ? (
          <OrgUserReference display={actorDisplay} size="sm" emphasizeName />
        ) : (
          <span className="font-semibold">Someone</span>
        )}
        <span>attached a</span>
        <span className="font-medium">{formatArtifactKindLong(artifact.type)}</span>
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
        {isMarkdown && artifact.body ? (
          <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-gray-700 dark:text-gray-300">
            {artifact.body}
          </p>
        ) : null}
      </div>
    </div>
  );
}

import { ExternalLink, FileText, GitBranch, GitPullRequest } from "lucide-react";
import { useState } from "react";

import type { OrgUserDisplayLookup } from "@/lib/orgUserDisplay";
import { safeExternalUrl } from "@/lib/safeExternalUrl";

import { OrgUserReference } from "../OrgUserReference";
import {
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
  formatPrArtifactLabel,
} from "../lib/workOrderArtifact";
import type { WorkOrderTimelineArtifact, WorkOrderTimelineEvent } from "../lib/workOrderTimelineEvents";
import { WorkOrderMarkdownArtifactDialog } from "../WorkOrderMarkdownArtifactDialog";
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

  const title = extractArtifactTitle(artifact.data);

  return (
    <div>
      <ArtifactAttribution
        artifactType={artifact.type}
        actorDisplay={actorDisplay}
        automationActor={event.actorAutomation}
      />
      <div className="mt-2 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-800 dark:border-gray-700/70 dark:bg-gray-900/40 dark:text-gray-200">
        <ArtifactBodyContent artifact={artifact} title={title} />
      </div>
    </div>
  );
}

function ArtifactBodyContent({ artifact, title }: { artifact: WorkOrderTimelineArtifact; title: string | undefined }) {
  switch (artifact.type) {
    case "pr":
      return <PrArtifactBody artifact={artifact} title={title} />;
    case "branch":
      return <BranchArtifactBody artifact={artifact} />;
    case "markdown":
    default:
      return <MarkdownArtifactBody artifact={artifact} title={title} />;
  }
}

function PrArtifactBody({ artifact, title }: { artifact: WorkOrderTimelineArtifact; title: string | undefined }) {
  const safeUrl = safeExternalUrl(extractArtifactUrl(artifact.data));
  const prLabel = formatPrArtifactLabel(artifact.data);
  const label = prLabel || title || safeUrl || "Pull request";

  if (!safeUrl) {
    return <p className="font-medium">{label}</p>;
  }
  return (
    <a
      href={safeUrl}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex max-w-full items-center gap-2 font-medium text-violet-700 hover:underline dark:text-violet-300"
    >
      <GitPullRequest className="h-4 w-4" aria-hidden />
      <span className="truncate">{label}</span>
      <ExternalLink className="h-3 w-3 shrink-0" aria-hidden />
    </a>
  );
}

function BranchArtifactBody({ artifact }: { artifact: WorkOrderTimelineArtifact }) {
  const label = extractArtifactName(artifact.data) || "Branch";

  return (
    <p className="inline-flex max-w-full items-center gap-2 font-medium">
      <GitBranch className="h-4 w-4 text-sky-500" aria-hidden />
      <span className="truncate">{label}</span>
    </p>
  );
}

function MarkdownArtifactBody({ artifact, title }: { artifact: WorkOrderTimelineArtifact; title: string | undefined }) {
  const [open, setOpen] = useState(false);
  const body = extractArtifactMarkdownBody(artifact.data) ?? "";
  const label = title || "Note";

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex max-w-full items-center gap-2 font-medium text-violet-700 hover:underline dark:text-violet-300"
      >
        <FileText className="h-4 w-4" aria-hidden />
        <span className="truncate">{label}</span>
      </button>
      <WorkOrderMarkdownArtifactDialog open={open} onClose={() => setOpen(false)} title={label} body={body} />
    </>
  );
}

function ArtifactAttribution({
  artifactType,
  actorDisplay,
  automationActor,
}: {
  artifactType: WorkOrderTimelineArtifact["type"];
  actorDisplay: ReturnType<OrgUserDisplayLookup>;
  automationActor: WorkOrderTimelineEvent["actorAutomation"];
}) {
  return (
    <p className="flex flex-wrap items-center gap-x-1 gap-y-1 text-sm text-gray-900 dark:text-gray-100">
      {actorDisplay ? (
        <OrgUserReference display={actorDisplay} size="sm" emphasizeName />
      ) : automationActor ? (
        <TimelineAutomationActor actor={automationActor} />
      ) : (
        <span className="font-semibold">Someone</span>
      )}
      <span>attached a</span>
      <span className="font-medium">{formatArtifactKindLong(artifactType)}</span>
      {automationActor && !actorDisplay ? (
        <span className="ml-1 inline-flex items-center gap-1 rounded border border-violet-200 bg-violet-50 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-violet-700 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-200">
          Automation
        </span>
      ) : null}
    </p>
  );
}

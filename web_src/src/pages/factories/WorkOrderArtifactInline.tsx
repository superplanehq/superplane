import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import {
  ExternalLink,
  FileText,
  GitBranch,
  GitMerge,
  GitPullRequest,
  GitPullRequestClosed,
  GitPullRequestDraft,
  Link as LinkIcon,
} from "lucide-react";
import { useState } from "react";

import {
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
  extractPrArtifactState,
  formatPrArtifactLabel,
  type PrArtifactState,
} from "./lib/workOrderArtifact";
import { WorkOrderMarkdownArtifactDialog } from "./WorkOrderMarkdownArtifactDialog";

export interface WorkOrderArtifactPresentation {
  id?: string;
  type: string;
  data?: Record<string, unknown>;
}

interface WorkOrderArtifactInlineProps {
  artifact: WorkOrderArtifactPresentation;
  className?: string;
}

const artifactInlineClassName =
  "inline-flex min-w-0 max-w-full items-center gap-1.5 text-[13px] font-medium tracking-[-0.01em] text-foreground";

export function WorkOrderArtifactInline({ artifact, className }: WorkOrderArtifactInlineProps) {
  const kind = normalizeArtifactKind(artifact.type);
  const safeUrl = safeExternalUrl(extractArtifactUrl(artifact.data));

  if (kind === "markdown") {
    return <MarkdownArtifactInline artifact={artifact} className={className} />;
  }

  const { icon: Icon, label, iconClassName } = artifactLinkPresentation(kind, artifact);
  const content = (
    <>
      <Icon className={cn("size-3.5 shrink-0", iconClassName ?? "text-muted-foreground")} aria-hidden />
      <span className="truncate">{label}</span>
      {safeUrl ? <ExternalLink className="size-3 shrink-0 text-muted-foreground" aria-hidden /> : null}
    </>
  );
  const classes = cn(artifactInlineClassName, className);

  if (!safeUrl) {
    return <span className={classes}>{content}</span>;
  }

  return (
    <a href={safeUrl} target="_blank" rel="noopener noreferrer" className={cn(classes, "hover:underline")}>
      {content}
    </a>
  );
}

function MarkdownArtifactInline({
  artifact,
  className,
}: {
  artifact: WorkOrderArtifactPresentation;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const label = extractArtifactTitle(artifact.data) ?? extractArtifactName(artifact.data) ?? "Note";
  const body = extractArtifactMarkdownBody(artifact.data) ?? "";

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={cn(
          artifactInlineClassName,
          "cursor-pointer justify-start border-0 bg-transparent p-0 text-left hover:underline",
          className,
        )}
      >
        <FileText className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
        <span className="truncate">{label}</span>
      </button>
      <WorkOrderMarkdownArtifactDialog open={open} onClose={() => setOpen(false)} title={label} body={body} />
    </>
  );
}

// GitHub-style icon + color per PR lifecycle state. `open` is also the
// back-compat default for artifacts attached before `state` existed (or
// carrying an unrecognized value) — same look as today, zero regression.
const PR_STATE_PRESENTATION: Record<PrArtifactState, { icon: typeof GitPullRequest; className: string }> = {
  open: { icon: GitPullRequest, className: "text-emerald-600 dark:text-emerald-400" },
  draft: { icon: GitPullRequestDraft, className: "text-muted-foreground" },
  closed: { icon: GitPullRequestClosed, className: "text-red-600 dark:text-red-400" },
  merged: { icon: GitMerge, className: "text-purple-600 dark:text-purple-400" },
};

function artifactLinkPresentation(
  kind: string,
  artifact: WorkOrderArtifactPresentation,
): { icon: typeof FileText; label: string; iconClassName?: string } {
  const title = extractArtifactTitle(artifact.data);
  const name = extractArtifactName(artifact.data);
  const url = extractArtifactUrl(artifact.data);

  switch (kind) {
    case "pr": {
      const { icon, className } = PR_STATE_PRESENTATION[extractPrArtifactState(artifact.data) ?? "open"];
      return {
        icon,
        iconClassName: className,
        label: firstLabel(formatPrArtifactLabel(artifact.data), title, name, compactUrlLabel(url), "Pull request"),
      };
    }
    case "branch":
      return { icon: GitBranch, label: firstLabel(name, title, compactUrlLabel(url), "Branch") };
    case "link":
    case "url":
    case "preview":
      return { icon: LinkIcon, label: firstLabel(name, title, compactUrlLabel(url), "Link") };
    default:
      return {
        icon: url ? LinkIcon : FileText,
        label: firstLabel(title, name, compactUrlLabel(url), "Artifact"),
      };
  }
}

function firstLabel(...labels: Array<string | undefined>): string {
  return labels.find((label) => Boolean(label?.trim())) ?? "Artifact";
}

function normalizeArtifactKind(type: string): string {
  return type.replace(/^TYPE_/i, "").toLowerCase();
}

function compactUrlLabel(value: string | undefined): string | undefined {
  const safeUrl = safeExternalUrl(value);
  if (!safeUrl) {
    return undefined;
  }

  try {
    const parsed = new URL(safeUrl);
    const lastSegment = parsed.pathname.split("/").filter(Boolean).at(-1);
    return lastSegment || parsed.hostname;
  } catch {
    return safeUrl;
  }
}

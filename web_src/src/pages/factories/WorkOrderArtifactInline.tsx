import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { ExternalLink, FileText, GitBranch, Link as LinkIcon } from "lucide-react";
import { useState } from "react";

import {
  type ArtifactData,
  branchTreeUrl,
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
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

  if (kind === "markdown") {
    return <MarkdownArtifactInline artifact={artifact} className={className} />;
  }

  const safeUrl = safeExternalUrl(artifactLinkUrl(kind, artifact.data));
  const { icon: Icon, label, fullLabel, iconClassName } = artifactLinkPresentation(kind, artifact);
  const content = (
    <>
      <Icon className={cn("size-3.5 shrink-0", iconClassName ?? "text-muted-foreground")} aria-hidden />
      <span className="truncate" title={fullLabel === label ? undefined : fullLabel}>
        {label}
      </span>
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

/** Branch names run long enough to push everything else off the row. */
const BRANCH_LABEL_MAX_CHARS = 25;

type ArtifactLinkPresentation = {
  icon: typeof FileText;
  label: string;
  fullLabel: string;
  iconClassName?: string;
};

function artifactLinkPresentation(kind: string, artifact: WorkOrderArtifactPresentation): ArtifactLinkPresentation {
  const title = extractArtifactTitle(artifact.data);
  const name = extractArtifactName(artifact.data);
  const url = extractArtifactUrl(artifact.data);

  switch (kind) {
    case "branch": {
      const branch = firstLabel(name, title, compactUrlLabel(url), "Branch");
      return { icon: GitBranch, label: capLabel(branch, BRANCH_LABEL_MAX_CHARS), fullLabel: branch };
    }
    case "link":
    case "url":
    case "preview":
      return presentation(LinkIcon, firstLabel(name, title, compactUrlLabel(url), "Link"));
    default:
      return presentation(url ? LinkIcon : FileText, firstLabel(title, name, compactUrlLabel(url), "Artifact"));
  }
}

function artifactLinkUrl(kind: string, data: ArtifactData): string | undefined {
  const url = extractArtifactUrl(data);
  if (url || kind !== "branch") {
    return url;
  }
  return branchTreeUrl(data);
}

function presentation(icon: typeof FileText, label: string, iconClassName?: string): ArtifactLinkPresentation {
  return { icon, label, fullLabel: label, iconClassName };
}

function capLabel(label: string, maxChars: number): string {
  if (label.length <= maxChars) {
    return label;
  }
  return `${label.slice(0, maxChars)}…`;
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

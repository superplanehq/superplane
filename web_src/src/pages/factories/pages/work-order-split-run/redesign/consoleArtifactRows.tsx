import { Button } from "@/components/ui/button";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight, Download, ExternalLink, FileText, GitBranch, Link2 } from "lucide-react";
import type { ReactNode } from "react";

import {
  artifactSizeLabel,
  extractArtifactContentType,
  extractArtifactMarkdownBody,
  fileArtifactIcon,
  toArtifactDataRecord,
} from "../../../lib/workOrderArtifact";
import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import type { AutomationStage } from "./automationsViewModel";
import {
  artifactLabel,
  artifactOpenHref,
  artifactsPageCount,
  consoleArtifactKind,
  isExpandableArtifact,
  isTaskDocument,
  type ConsoleArtifactKind,
  type StageArtifact,
} from "./consolePages";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

const ARTIFACT_ROW_MAIN =
  "group flex min-w-0 flex-1 items-center gap-1.5 py-1 text-[12px] font-medium text-foreground";

/**
 * Everything the run produced, on one page. Same flat list as Agent
 * runs: divider rows, not a pill or box per artifact. Documents and
 * media expand in place. Links and other files open in a new tab.
 */
export function ArtifactsPage({ stage, taskDocument }: { stage: AutomationStage; taskDocument?: ReactNode }) {
  const artifacts = stage.outputs.artifacts;
  const single = artifactsPageCount(stage) === 1;
  return (
    <div className="flex flex-col divide-y divide-border">
      {stage.outputs.pullRequests.map((pullRequest) => (
        <div key={pullRequest.id} className="py-1.5 first:pt-0 last:pb-0">
          <WorkOrderPullRequestInline pullRequest={pullRequest} showTitle className="text-[12px]" />
        </div>
      ))}
      {artifacts.map((artifact) => (
        <ArtifactRow
          key={artifact.id ?? artifactLabel(artifact)}
          artifact={artifact}
          defaultOpen={single}
          body={isTaskDocument(stage, artifact) ? taskDocument : undefined}
        />
      ))}
    </div>
  );
}

function ArtifactRow({
  artifact,
  defaultOpen,
  body,
}: {
  artifact: StageArtifact;
  defaultOpen: boolean;
  body?: ReactNode;
}) {
  const kind = consoleArtifactKind(artifact);
  const name = artifactLabel(artifact);
  const href = safeExternalUrl(artifactOpenHref(artifact)) ?? undefined;
  const Icon = artifactKindIcon(kind, artifact);
  const size = artifactSizeLabel(toArtifactDataRecord(artifact.data));
  if (isExpandableArtifact(kind)) {
    return (
      <ExpandableArtifactRow
        name={name}
        icon={<Icon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />}
        defaultOpen={defaultOpen}
        action={
          <ArtifactMeta size={size}>{expandableArtifactAction(kind, name, artifact, href)}</ArtifactMeta>
        }
      >
        {expandableArtifactBody(kind, name, artifact, href, body)}
      </ExpandableArtifactRow>
    );
  }
  return (
    <div className="flex items-center gap-1.5 py-1.5 first:pt-0 last:pb-0">
      {href ? (
        <a href={href} target="_blank" rel="noopener noreferrer" className={ARTIFACT_ROW_MAIN}>
          <Icon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span className="truncate">{name}</span>
        </a>
      ) : (
        <div className={ARTIFACT_ROW_MAIN}>
          <Icon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span className="truncate">{name}</span>
        </div>
      )}
      {href ? (
        <ArtifactMeta size={kind === "file" ? size : undefined}>
          <OpenableArtifactAction kind={kind} name={name} href={href} />
        </ArtifactMeta>
      ) : null}
    </div>
  );
}

function ArtifactMeta({ size, children }: { size?: string; children: ReactNode }) {
  if (!size && !children) {
    return null;
  }
  return (
    <div className="flex shrink-0 items-center gap-1">
      {size ? <span className={cn(META_TEXT_CLASSNAME, "tabular-nums")}>{size}</span> : null}
      {children}
    </div>
  );
}

function OpenableArtifactAction({ kind, name, href }: { kind: ConsoleArtifactKind; name: string; href: string }) {
  if (kind === "file") {
    return (
      <ArtifactIconLink href={href} download={name} label={`Download ${name}`}>
        <Download className="size-3.5" aria-hidden />
      </ArtifactIconLink>
    );
  }
  return (
    <ArtifactIconLink href={href} label={`Open ${name} in a new tab`}>
      <ExternalLink className="size-3.5" aria-hidden />
    </ArtifactIconLink>
  );
}

function ExpandableArtifactRow({
  name,
  icon,
  defaultOpen,
  action,
  children,
}: {
  name: string;
  icon: ReactNode;
  defaultOpen: boolean;
  action: ReactNode;
  children: ReactNode;
}) {
  return (
    <Collapsible defaultOpen={defaultOpen} className="py-1.5 first:pt-0 last:pb-0">
      <div className="flex items-center gap-1.5">
        <CollapsibleTrigger className={cn(ARTIFACT_ROW_MAIN, "cursor-pointer")}>
          <ChevronRight
            className="size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90"
            aria-hidden
          />
          {icon}
          <span className="truncate">{name}</span>
        </CollapsibleTrigger>
        {action}
      </div>
      {children ? (
        <CollapsibleContent className="space-y-3 py-2 pl-5 text-[12.5px] leading-5 text-muted-foreground">
          {children}
        </CollapsibleContent>
      ) : null}
    </Collapsible>
  );
}

function expandableArtifactBody(
  kind: ConsoleArtifactKind,
  name: string,
  artifact: StageArtifact,
  href: string | undefined,
  body: ReactNode,
): ReactNode {
  if (kind === "markdown") {
    return (
      body ?? (
        <MarkdownContent
          content={extractArtifactMarkdownBody(toArtifactDataRecord(artifact.data)) ?? ""}
          variant="workspace"
        />
      )
    );
  }
  if (!href) {
    return null;
  }
  if (kind === "video") {
    return <video controls preload="metadata" src={href} className="max-h-64 w-full" />;
  }
  return <img src={href} alt={name} loading="lazy" className="max-h-64 rounded-sm" />;
}

function expandableArtifactAction(
  kind: ConsoleArtifactKind,
  name: string,
  artifact: StageArtifact,
  href?: string,
): ReactNode {
  if (kind === "markdown") {
    const markdown = extractArtifactMarkdownBody(toArtifactDataRecord(artifact.data)) ?? "";
    const filename = name.endsWith(".md") ? name : `${name}.md`;
    return (
      <Button
        type="button"
        size="icon-xs"
        variant="ghost"
        className="size-7 shrink-0"
        aria-label={`Download ${filename}`}
        onClick={() => downloadTextFile(filename, markdown)}
      >
        <Download className="size-3.5" aria-hidden />
      </Button>
    );
  }
  if (!href) {
    return null;
  }
  return (
    <ArtifactIconLink href={href} download={name} label={`Download ${name}`}>
      <Download className="size-3.5" aria-hidden />
    </ArtifactIconLink>
  );
}

function ArtifactIconLink({
  href,
  download,
  label,
  children,
}: {
  href: string;
  download?: string;
  label: string;
  children: ReactNode;
}) {
  return (
    <Button type="button" size="icon-xs" variant="ghost" className="size-7 shrink-0" asChild>
      <a
        href={href}
        download={download}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={label}
      >
        {children}
      </a>
    </Button>
  );
}

function artifactKindIcon(kind: ConsoleArtifactKind, artifact: StageArtifact) {
  if (kind === "link") {
    return Link2;
  }
  if (kind === "branch") {
    return GitBranch;
  }
  if (kind === "markdown") {
    return FileText;
  }
  return fileArtifactIcon(extractArtifactContentType(toArtifactDataRecord(artifact.data)));
}

function downloadTextFile(filename: string, body: string) {
  const url = URL.createObjectURL(new Blob([body], { type: "text/markdown" }));
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

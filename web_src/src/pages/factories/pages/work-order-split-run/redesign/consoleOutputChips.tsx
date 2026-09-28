import { Badge, type BadgeProps } from "@/components/reui/badge";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { Download, FileText, GitBranch, Link2 } from "lucide-react";
import { useState, type ReactNode } from "react";

import { fileArtifactIcon } from "../../../lib/workOrderArtifact";
import {
  branchTreeUrl,
  extractArtifactContentType,
  extractArtifactFilename,
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
  toArtifactDataRecord,
} from "../../../lib/workOrderArtifact";
import { formatCheckScore, type WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { WorkOrderMarkdownArtifactDialog } from "../../../WorkOrderMarkdownArtifactDialog";
import { outputCountLabel, stepOutputSummary } from "./consoleCardText";
import type { AutomationStage } from "./automationsViewModel";

export function StepOutputCounts({ stage }: { stage: AutomationStage }) {
  const summary = stepOutputSummary(stage);
  if (summary.artifactCount === 0 && summary.checkCount === 0) {
    return null;
  }
  return (
    <div className="flex min-w-0 items-center gap-1.5">
      {summary.artifactCount > 0 ? (
        <OutputCountHover
          stageId={stage.id}
          kind="artifacts"
          label={outputCountLabel(summary.artifactCount, "artifact", "artifacts")}
        >
          <div className="flex flex-col gap-2">
            {stage.outputs.artifacts.map((artifact) => (
              <ArtifactChip key={artifact.id} artifact={artifact} />
            ))}
          </div>
        </OutputCountHover>
      ) : null}
      {summary.checkCount > 0 ? (
        <OutputCountHover
          stageId={stage.id}
          kind="checks"
          label={outputCountLabel(summary.checkCount, "check", "checks")}
        >
          <div className="flex flex-col gap-2">
            {stage.checks.map((check) => (
              <CheckBadgeRow key={check.id} check={check} />
            ))}
          </div>
        </OutputCountHover>
      ) : null}
    </div>
  );
}

function OutputCountHover({
  stageId,
  kind,
  label,
  children,
}: {
  stageId: string;
  kind: "artifacts" | "checks";
  label: string;
  children: ReactNode;
}) {
  return (
    <HoverCard openDelay={0} closeDelay={80}>
      <HoverCardTrigger asChild>
        <button type="button" data-testid={`redesign-console-${kind}-trigger-${stageId}`}>
          <Badge variant="outline" className="h-5 px-1.5 text-[12px] font-normal">
            {label}
          </Badge>
        </button>
      </HoverCardTrigger>
      <HoverCardContent
        align="start"
        className="w-auto max-w-sm p-3"
        data-testid={`redesign-console-${kind}-hover-${stageId}`}
      >
        {children}
      </HoverCardContent>
    </HoverCard>
  );
}

export function ArtifactChip({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const kind = (artifact.type ?? "").replace(/^TYPE_/i, "").toLowerCase();
  if (kind === "markdown") {
    return <MarkdownArtifactChip artifact={artifact} />;
  }
  if (kind === "branch") {
    return <BranchArtifactChip artifact={artifact} />;
  }
  if (kind === "link") {
    return <LinkArtifactChip artifact={artifact} />;
  }
  return <FileArtifactChip artifact={artifact} />;
}

function FileArtifactChip({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const data = toArtifactDataRecord(artifact.data);
  const name = extractArtifactFilename(data) ?? extractArtifactName(data) ?? extractArtifactTitle(data) ?? "File";
  const size = typeof data?.size === "string" ? data.size : undefined;
  const url = safeExternalUrl(extractArtifactUrl(data)) ?? undefined;
  const Icon = fileArtifactIcon(extractArtifactContentType(data));
  return (
    <ArtifactActionGroup
      icon={<Icon aria-hidden />}
      name={name}
      size={size}
      openHref={url}
      openLabel={`Open ${name}`}
      downloadHref={url}
      downloadName={name}
    />
  );
}

function MarkdownArtifactChip({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const data = toArtifactDataRecord(artifact.data);
  const name = extractArtifactTitle(data) ?? extractArtifactName(data) ?? "Note";
  const size = typeof data?.size === "string" ? data.size : undefined;
  const body = extractArtifactMarkdownBody(data) ?? "";
  const [open, setOpen] = useState(false);
  return (
    <>
      <ArtifactActionGroup
        icon={<FileText aria-hidden />}
        name={name}
        size={size}
        onOpen={() => setOpen(true)}
        openLabel={`Open ${name}`}
        onDownload={() => downloadTextFile(name.endsWith(".md") ? name : `${name}.md`, body)}
        downloadName={name}
      />
      <WorkOrderMarkdownArtifactDialog open={open} onClose={() => setOpen(false)} title={name} body={body} />
    </>
  );
}

function BranchArtifactChip({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const data = toArtifactDataRecord(artifact.data);
  const name = extractArtifactName(data) ?? extractArtifactTitle(data) ?? "Branch";
  const href = safeExternalUrl(extractArtifactUrl(data) ?? branchTreeUrl(data)) ?? undefined;
  return <SingleOpenChip icon={<GitBranch aria-hidden />} name={name} href={href} />;
}

function LinkArtifactChip({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const data = toArtifactDataRecord(artifact.data);
  const name = extractArtifactTitle(data) ?? extractArtifactName(data) ?? extractArtifactUrl(data) ?? "Link";
  const href = safeExternalUrl(extractArtifactUrl(data)) ?? undefined;
  return <SingleOpenChip icon={<Link2 aria-hidden />} name={name} href={href} />;
}

function SingleOpenChip({ icon, name, href }: { icon: ReactNode; name: string; href?: string }) {
  const label = (
    <>
      {icon}
      {name}
    </>
  );
  if (!href) {
    return (
      <Button type="button" variant="outline" size="sm" className={FILE_CHIP_BUTTON}>
        {label}
      </Button>
    );
  }
  return (
    <Button type="button" variant="outline" size="sm" className={FILE_CHIP_BUTTON} asChild>
      <a href={href} target="_blank" rel="noopener noreferrer">
        {label}
      </a>
    </Button>
  );
}

const CHIP_GROUP =
  "[&>*:first-child]:rounded-l-md! [&>*:last-child]:rounded-r-md! [&>*:not(:first-child)]:rounded-l-none! [&>*:not(:last-child)]:rounded-r-none!";

const FILE_CHIP_BUTTON =
  "h-7 gap-1.5 rounded-md px-2 text-xs font-normal shadow-none [&_svg]:size-3.5 dark:bg-background dark:hover:bg-accent";

function ArtifactActionGroup({
  icon,
  name,
  size,
  openHref,
  onOpen,
  openLabel,
  downloadHref,
  downloadName,
  onDownload,
}: {
  icon: ReactNode;
  name: string;
  size?: string;
  openHref?: string;
  onOpen?: () => void;
  openLabel: string;
  downloadHref?: string;
  downloadName: string;
  onDownload?: () => void;
}) {
  const label = (
    <>
      {icon}
      {name}
      {size ? <span className="opacity-60">({size})</span> : null}
    </>
  );
  return (
    <ButtonGroup className={CHIP_GROUP}>
      {openHref ? (
        <Button type="button" variant="outline" size="sm" className={FILE_CHIP_BUTTON} asChild>
          <a href={openHref} target="_blank" rel="noopener noreferrer" aria-label={openLabel}>
            {label}
          </a>
        </Button>
      ) : (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className={FILE_CHIP_BUTTON}
          onClick={onOpen}
          aria-label={openLabel}
        >
          {label}
        </Button>
      )}
      {downloadHref || onDownload ? (
        downloadHref ? (
          <Button type="button" variant="outline" size="icon-xs" className={cn(FILE_CHIP_BUTTON, "w-7 px-0")} asChild>
            <a href={downloadHref} download={downloadName} aria-label={`Download ${downloadName}`}>
              <Download aria-hidden />
            </a>
          </Button>
        ) : (
          <Button
            type="button"
            variant="outline"
            size="icon-xs"
            className={cn(FILE_CHIP_BUTTON, "w-7 px-0")}
            aria-label={`Download ${downloadName}`}
            onClick={onDownload}
          >
            <Download aria-hidden />
          </Button>
        )
      ) : null}
    </ButtonGroup>
  );
}

function downloadTextFile(filename: string, body: string) {
  const url = URL.createObjectURL(new Blob([body], { type: "text/markdown" }));
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

const CHECK_BADGE: Record<
  WorkOrderCheckPresentation["level"],
  { variant: BadgeProps["variant"]; dotClassName: string }
> = {
  positive: { variant: "success-light", dotClassName: "bg-success" },
  neutral: { variant: "secondary", dotClassName: "bg-muted-foreground" },
  caution: { variant: "warning-light", dotClassName: "bg-warning" },
  critical: { variant: "destructive-light", dotClassName: "bg-destructive" },
};

export function CheckBadgeRow({ check }: { check: WorkOrderCheckPresentation }) {
  const tone = CHECK_BADGE[check.level];
  const score = formatCheckScore(check);
  const scoreLabel = `${score.value}${score.scale}`;
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <Badge variant={tone.variant}>{check.name}</Badge>
      {scoreLabel ? <DotBadge label={scoreLabel} dotClassName={tone.dotClassName} /> : null}
    </div>
  );
}

/** Status dot from the timeline-2 block. */
function DotBadge({ label, dotClassName }: { label: string; dotClassName: string }) {
  return (
    <Badge variant="outline" className="gap-1.5">
      <span className={cn("size-1.5 rounded-full", dotClassName)} aria-hidden />
      {label}
    </Badge>
  );
}

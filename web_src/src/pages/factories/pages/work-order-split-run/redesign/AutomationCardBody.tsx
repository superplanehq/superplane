import { Badge } from "@/components/reui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight, CircleStop, Download, FileText, History, RotateCw } from "lucide-react";
import { useState, type ReactNode } from "react";

import type { FilesFile } from "@/api-client";

import { extractArtifactMarkdownBody, extractArtifactUrl, toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { formatCheckScore, workOrderCheckStatus, type WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { WorkOrderCheckAnalysis } from "../../../WorkOrderCheckDialog";
import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import type { SplitRunPhase } from "../splitRunMocks";
import type { SplitRunSource } from "../splitRunSource";
import { WorkOrderSplitRunDescription } from "../WorkOrderSplitRunDescription";
import { WorkOrderSplitRunSource } from "../WorkOrderSplitRunSource";
import type { AutomationStage, ConsoleAutomation } from "./automationsViewModel";
import { runFooterLine } from "./consoleCardText";
import { ArtifactChip } from "./consoleOutputChips";
import {
  artifactLabel,
  artifactsPageCount,
  consoleArtifactKind,
  consolePages,
  isTaskDocument,
  type ConsolePageId,
  type StageArtifact,
} from "./consolePages";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

// Line tabs on the shadcn Tabs primitives, after ReUI c-tabs-2.
const PANE_TABS_LIST =
  "h-auto w-full justify-start gap-1 overflow-x-auto rounded-none border-b border-border bg-transparent p-0 dark:bg-transparent";
const PANE_TAB =
  "h-7 flex-none gap-1.5 rounded-none border-0 border-b-2 border-transparent px-2 text-[12px] font-medium text-muted-foreground shadow-none " +
  "data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none " +
  "dark:data-[state=active]:border-primary dark:data-[state=active]:bg-transparent";

const PAGE_LABEL: Record<ConsolePageId, string> = {
  agent: "Agent log",
  artifacts: "Artifacts",
  checks: "Checks",
};

export function AutomationCardBody({
  automation,
  phase,
  organizationId,
  factoryId,
  orderId,
  taskDescription,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  source,
  files,
  onStop,
  onRetry,
  actionBusy,
  onOpen,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  taskDescription?: string;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  /** Where the task came from. The Backlog creation card shows it under the task text. */
  source?: SplitRunSource;
  files?: FilesFile[];
  onStop?: () => void;
  onRetry?: () => void;
  actionBusy: boolean;
  onOpen: () => void;
}) {
  const { latest, runs } = automation;
  const pages = consolePages(latest, phase);
  const [chosen, setChosen] = useState<ConsolePageId>();
  const active = chosen && pages.includes(chosen) ? chosen : pages[0];
  return (
    <div className="space-y-3">
      <StageDescription stage={latest} />
      <CardPageTabs pages={pages} active={active} stage={latest} onChange={setChosen} />
      {/* The log stays mounted so live steps and spend keep streaming. */}
      {pages.includes("agent") ? (
        <div className={cn(active !== "agent" && "hidden")}>
          <LiveAgentSteps
            stage={latest}
            phase={phase}
            organizationId={organizationId}
            emptyNote="No agent log for this run."
          />
        </div>
      ) : null}
      {active === "artifacts" ? (
        <ArtifactsPage
          stage={latest}
          taskDocument={
            latest.id === "backlog" ? (
              <CreationTaskDocument
                description={taskDescription}
                canEdit={canEditDescription}
                busy={descriptionBusy}
                onSave={onDescriptionSave}
                files={files}
                organizationId={organizationId}
                factoryId={factoryId}
                orderId={orderId}
              />
            ) : undefined
          }
        />
      ) : null}
      {active === "checks" ? <ChecksPage checks={latest.checks} /> : null}
      <CreationSource stageId={latest.id} source={source} />
      <CardRunFooter
        stage={latest}
        runCount={runs.length}
        actionBusy={actionBusy}
        onOpen={onOpen}
        onRetry={onRetry}
        onStop={onStop}
      />
    </div>
  );
}

function CardPageTabs({
  pages,
  active,
  stage,
  onChange,
}: {
  pages: ConsolePageId[];
  active?: ConsolePageId;
  stage: AutomationStage;
  onChange: (page: ConsolePageId) => void;
}) {
  if (pages.length === 0 || !active) {
    return null;
  }
  return (
    <Tabs value={active} onValueChange={(next) => onChange(next as ConsolePageId)}>
      <TabsList aria-label="Run details" className={PANE_TABS_LIST}>
        {pages.map((page) => (
          <TabsTrigger key={page} value={page} className={PANE_TAB}>
            {PAGE_LABEL[page]}
            <PageTabDetail page={page} stage={stage} />
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}

function CreationTaskDocument({
  description,
  canEdit,
  busy,
  onSave,
  files,
  organizationId,
  factoryId,
  orderId,
}: {
  description?: string;
  canEdit: boolean;
  busy: boolean;
  onSave?: (next: string) => void | Promise<void>;
  files?: FilesFile[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
}) {
  return (
    <div data-testid="redesign-console-task-description">
      <WorkOrderSplitRunDescription
        description={description ?? ""}
        canEdit={canEdit}
        busy={busy}
        collapsible={false}
        onSave={onSave}
        files={files}
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
      />
    </div>
  );
}

function CreationSource({ stageId, source }: { stageId: string; source?: SplitRunSource }) {
  if (stageId !== "backlog" || !source) {
    return null;
  }
  return (
    <div data-testid="redesign-console-card-source">
      <span className="text-[12px] font-medium text-muted-foreground">Source</span>
      <WorkOrderSplitRunSource source={source} />
    </div>
  );
}

function CardRunFooter({
  stage,
  runCount,
  actionBusy,
  onOpen,
  onRetry,
  onStop,
}: {
  stage: AutomationStage;
  runCount: number;
  actionBusy: boolean;
  onOpen: () => void;
  onRetry?: () => void;
  onStop?: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t pt-3">
      <span className={cn(META_TEXT_CLASSNAME, "min-w-0")}>{runFooterLine(stage)}</span>
      <div className="ms-auto flex shrink-0 items-center gap-1.5">
        {runCount > 1 ? (
          <Button size="sm" variant="ghost" className="h-7 gap-1.5 px-2 text-[12px]" onClick={onOpen}>
            <History className="size-3.5" aria-hidden />
            View {runCount} runs
          </Button>
        ) : null}
        {onRetry ? (
          <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onRetry}>
            <RotateCw className="size-3.5" aria-hidden />
            Retry
          </Button>
        ) : null}
        {onStop ? (
          <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onStop}>
            <CircleStop className="size-3.5" aria-hidden />
            Stop
          </Button>
        ) : null}
      </div>
    </div>
  );
}

/** Type-specific tab detail: live dot, artifact count, or check values. */
function PageTabDetail({ page, stage }: { page: ConsolePageId; stage: AutomationStage }) {
  if (page === "agent") {
    return stage.status === "running" ? (
      <span className="size-1.5 animate-pulse rounded-full bg-primary" aria-hidden />
    ) : null;
  }
  if (page === "artifacts") {
    return <span className="tabular-nums opacity-60">{artifactsPageCount(stage)}</span>;
  }
  const shown = stage.checks.slice(0, 2);
  return (
    <>
      {shown.map((check) => (
        <CheckTabValue key={check.id} check={check} />
      ))}
      {stage.checks.length > shown.length ? (
        <span className="tabular-nums opacity-60">+{stage.checks.length - shown.length}</span>
      ) : null}
    </>
  );
}

/** The score itself sits in the tab, in its verdict color. */
function CheckTabValue({ check }: { check: WorkOrderCheckPresentation }) {
  const { value, scale } = formatCheckScore(check);
  return (
    <span className={cn("tabular-nums", workOrderCheckStatus(check).className)}>
      {value}
      {scale}
    </span>
  );
}

/** The stage description always shows above the tabs. It is context, not a page. */
function StageDescription({ stage }: { stage: AutomationStage }) {
  if (stage.id === "backlog" || !stage.description?.trim()) {
    return null;
  }
  return (
    <div className="text-[12.5px] leading-5 text-muted-foreground">
      <MarkdownContent content={stage.description} variant="workspace" />
    </div>
  );
}

/**
 * Everything the run produced, on one page: the pull request first, then
 * documents reading inline, link and file chips, and visual evidence
 * playing in the body. Nothing opens a popup.
 */
function ArtifactsPage({ stage, taskDocument }: { stage: AutomationStage; taskDocument?: ReactNode }) {
  const artifacts = stage.outputs.artifacts;
  const documents = artifacts.filter((artifact) => consoleArtifactKind(artifact) === "markdown");
  const media = artifacts.filter((artifact) => ["image", "video"].includes(consoleArtifactKind(artifact)));
  const chips = artifacts.filter((artifact) => !documents.includes(artifact) && !media.includes(artifact));
  const single = artifactsPageCount(stage) === 1;
  return (
    <div className="space-y-3">
      {stage.outputs.pullRequests.map((pullRequest) => (
        <WorkOrderPullRequestInline key={pullRequest.id} pullRequest={pullRequest} showTitle className="text-[12px]" />
      ))}
      {documents.map((artifact) => (
        <DocumentArtifact key={artifact.id ?? artifactLabel(artifact)} artifact={artifact} defaultOpen={single}>
          {isTaskDocument(stage, artifact) ? taskDocument : undefined}
        </DocumentArtifact>
      ))}
      {chips.length > 0 ? (
        <div className="flex flex-wrap items-center gap-2">
          {chips.map((artifact) => (
            <ArtifactChip key={artifact.id ?? artifactLabel(artifact)} artifact={artifact} />
          ))}
        </div>
      ) : null}
      {media.map((artifact) => (
        <MediaArtifact key={artifact.id ?? artifactLabel(artifact)} artifact={artifact} />
      ))}
    </div>
  );
}

/**
 * A produced document reads inline. The row toggles it; a lone document
 * starts open. `children` replaces the markdown body — the creation card
 * puts the editable task description there.
 */
function DocumentArtifact({
  artifact,
  defaultOpen,
  children,
}: {
  artifact: StageArtifact;
  defaultOpen: boolean;
  children?: ReactNode;
}) {
  const name = artifactLabel(artifact);
  const body = extractArtifactMarkdownBody(toArtifactDataRecord(artifact.data)) ?? "";
  const filename = name.endsWith(".md") ? name : `${name}.md`;
  return (
    <Collapsible defaultOpen={defaultOpen} className="rounded-md border">
      <div className="flex items-center gap-1 pr-1">
        <CollapsibleTrigger className="group flex min-w-0 flex-1 cursor-pointer items-center gap-1.5 px-2 py-1.5 text-[12px] font-medium text-foreground">
          <ChevronRight
            className="size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90"
            aria-hidden
          />
          <FileText className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span className="truncate">{name}</span>
        </CollapsibleTrigger>
        <Button
          type="button"
          size="icon-xs"
          variant="ghost"
          className="size-7 shrink-0"
          aria-label={`Download ${filename}`}
          onClick={() => downloadTextFile(filename, body)}
        >
          <Download className="size-3.5" aria-hidden />
        </Button>
      </div>
      <CollapsibleContent className="border-t px-3 py-2 text-[12.5px] leading-5 text-muted-foreground">
        {children ?? <MarkdownContent content={body} variant="workspace" />}
      </CollapsibleContent>
    </Collapsible>
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

/** Visual evidence renders in the body: images link out full size, videos play inline. */
function MediaArtifact({ artifact }: { artifact: StageArtifact }) {
  const data = toArtifactDataRecord(artifact.data);
  const url = safeExternalUrl(extractArtifactUrl(data));
  const name = artifactLabel(artifact);
  if (!url) {
    return <ArtifactChip artifact={artifact} />;
  }
  if (consoleArtifactKind(artifact) === "video") {
    return <video controls preload="metadata" src={url} className="max-h-64 w-full rounded-md border" />;
  }
  return (
    <a href={url} target="_blank" rel="noopener noreferrer" className="block w-fit" aria-label={`Open ${name}`}>
      <img src={url} alt={name} loading="lazy" className="max-h-64 rounded-md border" />
    </a>
  );
}

/** Every score on one page: value and verdict up front, the analysis on demand. */
function ChecksPage({ checks }: { checks: WorkOrderCheckPresentation[] }) {
  return (
    <div className="flex flex-col divide-y divide-border">
      {checks.map((check) => (
        <CheckSection key={check.id} check={check} defaultOpen={checks.length === 1} />
      ))}
    </div>
  );
}

function CheckSection({ check, defaultOpen }: { check: WorkOrderCheckPresentation; defaultOpen: boolean }) {
  const { value, scale } = formatCheckScore(check);
  const status = workOrderCheckStatus(check);
  const expandable = Boolean(check.analysis?.trim());
  const header = (
    <>
      <span className="text-[12px] font-medium text-foreground">{check.name}</span>
      <span className={cn("text-[13px] font-semibold tabular-nums", status.className)}>
        {value}
        {scale}
      </span>
      <Badge variant="outline" className={cn("border", status.badgeClassName)}>
        {status.label}
      </Badge>
    </>
  );
  return (
    <Collapsible defaultOpen={defaultOpen && expandable} className="py-2.5 first:pt-0 last:pb-0">
      {expandable ? (
        <CollapsibleTrigger className="group flex w-full cursor-pointer items-center gap-2 text-left">
          {header}
          <ChevronRight
            className="ms-auto size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90"
            aria-hidden
          />
        </CollapsibleTrigger>
      ) : (
        <div className="flex items-center gap-2">{header}</div>
      )}
      {check.summary ? <p className="mt-1.5 text-[12px] leading-5 text-muted-foreground">{check.summary}</p> : null}
      {expandable ? (
        <CollapsibleContent>
          <WorkOrderCheckAnalysis check={check} />
        </CollapsibleContent>
      ) : null}
    </Collapsible>
  );
}

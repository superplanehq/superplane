import { Frame, FrameDescription, FrameHeader, FramePanel, FrameTitle } from "@/components/reui/frame";
import {
  Timeline,
  TimelineContent,
  TimelineHeader,
  TimelineIndicator,
  TimelineItem,
  TimelineSeparator,
  TimelineTitle,
} from "@/components/reui/timeline";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Badge, type BadgeProps } from "@/components/reui/badge";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ButtonGroup } from "@/components/ui/button-group";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/ui/sheet";
import {
  ArrowLeft,
  ChevronRight,
  Download,
  FileText,
  GitBranch,
  CircleStop,
  History,
  Link2,
  Maximize2,
  RotateCw,
} from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";

import { OrgUserReference } from "../../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import { formatCheckScore, type WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import {
  extractArtifactContentType,
  extractArtifactFilename,
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
  toArtifactDataRecord,
  branchTreeUrl,
} from "../../../lib/workOrderArtifact";
import { WorkOrderMarkdownArtifactDialog } from "../../../WorkOrderMarkdownArtifactDialog";
import { fileArtifactIcon } from "../../../WorkOrderArtifactInline";
import { SplitRunCheckPills } from "../SplitRunReview";
import type { SplitRunFixture, SplitRunPhase, SplitRunPhaseStatus } from "../splitRunMocks";
import { splitRunLinkedArtifacts, splitRunPhaseRunHref } from "../splitRunPopupModel";
import { LiveAgentSteps } from "./LiveAgentSteps";
import {
  outputCountLabel,
  runFooterLine,
  runMetaLine,
  showDescriptionInBody,
  stepOutputSummary,
} from "./consoleCardText";
import {
  allStages,
  isConsoleTaskStage,
  outcomeSummary,
  stagesFromFixture,
  type AutomationStage,
} from "./automationsViewModel";
import { META_TEXT_CLASSNAME, formatClock } from "./redesignFormat";
import { StageStatusGlyph } from "./redesignShared";

/**
 * Variant B: automation console. Backlog, Implement, Verify, and Done sit
 * on a timeline. Each card is the latest run of one automation, seen
 * through its agent. The header carries status and this run's spend. The
 * body is the same marker list on a finished run and on a running one. A
 * running step is the live row. The footer carries start time, model, and
 * the way into the runs drawer. Canvas nodes are not shown on this tab.
 * Columns the task has not reached read "Not started". A sticky Frame
 * holds the task status, spend, checks, and outputs.
 */
export function AutomationsConsoleVariant({
  fixture,
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
  taskDescription,
  panelReview,
  canStopRun = false,
  actionBusy = false,
  onStopRun,
  onRerunStep,
}: {
  fixture: SplitRunFixture;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  /** Task description markdown for the creation card body. */
  taskDescription?: string;
  /** Decision note and actions for the summary panel. */
  panelReview?: ReactNode;
  /** True when this person can cancel a live canvas run. */
  canStopRun?: boolean;
  actionBusy?: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  /** Reruns a failed line step. The card footer Retry uses this. */
  onRerunStep?: (phase: SplitRunPhase) => void;
}) {
  const outcome = outcomeSummary(fixture);
  const groups = stagesFromFixture(fixture);
  const stages = allStages(groups);
  const columns = consoleColumns(groups, fixture.footer.run?.appId);
  const currentColumn = reachedColumns(columns);
  const [openAutomation, setOpenAutomation] = useState<ConsoleAutomation | null>(null);
  const [fullLogAutomation, setFullLogAutomation] = useState<ConsoleAutomation | null>(null);

  if (fullLogAutomation) {
    const phase = fixture.phases.find((entry) => entry.id === fullLogAutomation.latest.id);
    return (
      <ConsoleFullLog
        automation={fullLogAutomation}
        phase={phase}
        organizationId={organizationId}
        runHref={phase ? splitRunPhaseRunHref({ organizationId, factoryKey, orderNumber, lineId, phase }) : undefined}
        onBack={() => setFullLogAutomation(null)}
      />
    );
  }

  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_340px]" data-testid="redesign-console-variant">
      <section className="flex min-w-0 flex-col gap-4">
        <Timeline value={currentColumn} className="pl-1">
          {columns.map((column, index) => (
            <TimelineItem key={column.id} step={index + 1} data-testid={`redesign-console-column-${column.id}`}>
              <TimelineSeparator />
              <TimelineIndicator />
              <TimelineHeader className="flex items-center gap-2">
                <TimelineTitle>{column.title}</TimelineTitle>
                {column.automations.length > 0 ? (
                  <Badge variant="secondary" className="h-5 min-w-5 px-1.5">
                    {column.automations.length}
                  </Badge>
                ) : null}
              </TimelineHeader>
              <TimelineContent className="mt-2 flex flex-col gap-3 text-foreground">
                {column.automations.length === 0 ? (
                  <span className={META_TEXT_CLASSNAME}>Not started</span>
                ) : (
                  column.automations.map((automation) => (
                    <ConsoleAutomationCard
                      key={automation.id}
                      automation={automation}
                      phase={fixture.phases.find((phase) => phase.id === automation.latest.id)}
                      organizationId={organizationId}
                      factoryKey={factoryKey}
                      orderNumber={orderNumber}
                      lineId={lineId}
                      taskDescription={taskDescription}
                      defaultOpen={index + 1 === currentColumn}
                      canStopRun={canStopRun}
                      actionBusy={actionBusy}
                      onStopRun={onStopRun}
                      onRerunStep={onRerunStep}
                      onOpen={() => setOpenAutomation(automation)}
                      onFullLog={() => setFullLogAutomation(automation)}
                    />
                  ))
                )}
              </TimelineContent>
            </TimelineItem>
          ))}
        </Timeline>
      </section>
      <ConsoleSummaryPanel fixture={fixture} outcome={outcome} stages={stages} panelReview={panelReview} />
      <ConsoleRunsDrawer
        automation={openAutomation}
        open={openAutomation !== null}
        onOpenChange={(open) => !open && setOpenAutomation(null)}
        fixture={fixture}
        organizationId={organizationId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
        lineId={lineId}
      />
    </div>
  );
}

const CONSOLE_COLUMNS = [
  { id: "backlog", title: "Backlog", names: ["Backlog", "Analysis"] },
  { id: "implement", title: "Implement", names: ["Implement"] },
  { id: "verify", title: "Verify", names: [] },
  { id: "done", title: "Done", names: [] },
] as const;

/** One automation with every run it made for this task, newest first. */
interface ConsoleAutomation {
  id: string;
  name: string;
  latest: AutomationStage;
  runs: AutomationStage[];
}

interface ConsoleColumn {
  id: string;
  title: string;
  automations: ConsoleAutomation[];
}

/**
 * Task stages sit in the column named after them. The creation stage
 * sits in Backlog even when no automation ran. Pull request activity
 * sits in Verify, except the runs of the automation that closed the
 * task: those sit in Done.
 */
function consoleColumns(groups: ReturnType<typeof stagesFromFixture>, closerAppId?: string): ConsoleColumn[] {
  const pullRequestStages = groups.pullRequestGroups.flatMap((group) => group.stages).filter((stage) => stage.appId);
  const closedBy = (stage: AutomationStage) => Boolean(closerAppId) && stage.appId === closerAppId;
  return CONSOLE_COLUMNS.map((column) => {
    const stages =
      column.id === "verify"
        ? pullRequestStages.filter((stage) => !closedBy(stage))
        : column.id === "done"
          ? pullRequestStages.filter(closedBy)
          : groups.taskStages
              .filter(isConsoleTaskStage)
              .filter((stage) => (column.names as readonly string[]).includes(stage.name));
    return { id: column.id, title: column.title, automations: automationsFromStages(stages) };
  });
}

/** Timeline steps to fill: through the last column that has a run. */
function reachedColumns(columns: ConsoleColumn[]): number {
  const reached = columns.map((column) => column.automations.length > 0).lastIndexOf(true);
  return reached + 1;
}

function automationsFromStages(stages: AutomationStage[]): ConsoleAutomation[] {
  const byKey = new Map<string, AutomationStage[]>();
  for (const stage of stages) {
    const key = stage.appId || stage.componentName;
    const runs = byKey.get(key) ?? [];
    runs.push(stage);
    byKey.set(key, runs);
  }
  return [...byKey.entries()].map(([key, runs]) => {
    const newestFirst = [...runs].sort(
      (left, right) => Date.parse(right.startedAt ?? "") - Date.parse(left.startedAt ?? ""),
    );
    const name = newestFirst[0]?.componentName ?? key;
    return {
      id: automationDomId(name, key),
      name,
      latest: newestFirst[0],
      runs: newestFirst,
    };
  });
}

function automationDomId(name: string, key: string): string {
  const slug = `${name}-${key}`.toLowerCase().replace(/\W+/g, "-").replace(/^-|-$/g, "");
  return slug || "automation";
}

function ConsoleAutomationCard({
  automation,
  phase,
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
  taskDescription,
  defaultOpen,
  canStopRun,
  actionBusy,
  onStopRun,
  onRerunStep,
  onOpen,
  onFullLog,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  taskDescription?: string;
  /** True for cards in the column the task is currently in. */
  defaultOpen: boolean;
  canStopRun: boolean;
  actionBusy: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  onRerunStep?: (phase: SplitRunPhase) => void;
  onOpen: () => void;
  onFullLog: () => void;
}) {
  const { latest } = automation;
  const stopping = useStopRequested(latest.status, actionBusy);
  const shownStatus: SplitRunPhaseStatus = stopping.active && latest.status === "running" ? "cancelled" : latest.status;
  const shownPhase = phase && shownStatus !== phase.status ? { ...phase, status: shownStatus } : phase;
  const runHref = shownPhase
    ? splitRunPhaseRunHref({ organizationId, factoryKey, orderNumber, lineId, phase: shownPhase })
    : undefined;
  const stopRun =
    canStopRun && onStopRun && shownStatus === "running" && shownPhase?.appId && shownPhase.runId
      ? () => {
          stopping.request();
          onStopRun({ appId: shownPhase.appId ?? "", runId: shownPhase.runId ?? "" });
        }
      : undefined;
  const rerunStep =
    canStopRun && onRerunStep && shownStatus === "failed" && shownPhase?.stepIndex != null
      ? () => onRerunStep(shownPhase)
      : undefined;
  return (
    <Frame
      variant="default"
      spacing="sm"
      stacked
      dense
      className="[--frame-radius:var(--radius-lg)]"
      data-testid={`redesign-console-automation-${automation.id}`}
    >
      <Collapsible defaultOpen={defaultOpen || shownStatus === "running"} className="group/collapsible">
        <FrameHeader className="flex min-w-0 flex-row items-center gap-2 py-2">
          <CollapsibleTrigger
            className="flex min-w-0 shrink-0 items-center gap-2"
            aria-label={`Toggle ${automation.name} details`}
          >
            <StageStatusGlyph status={shownStatus} />
            <span className="shrink-0 text-[13px] font-medium text-foreground">{automation.name}</span>
          </CollapsibleTrigger>
          <StepOutputCounts stage={latest} />
          <span className={cn(META_TEXT_CLASSNAME, "ml-auto px-1.5 tabular-nums")}>{runMetaLine(latest)}</span>
          {latest.appId ? <FullLogButton onFullLog={onFullLog} /> : null}
          <CollapsibleTrigger
            className="inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
            tabIndex={-1}
            aria-hidden
          >
            <ChevronRight
              className="size-4 shrink-0 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
              aria-hidden
            />
          </CollapsibleTrigger>
        </FrameHeader>
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            <AutomationCardBody
              automation={automation}
              phase={shownPhase}
              organizationId={organizationId}
              taskDescription={taskDescription}
              runHref={runHref}
              onStop={stopRun}
              onRetry={rerunStep}
              actionBusy={actionBusy}
              onOpen={onOpen}
            />
          </FramePanel>
        </CollapsibleContent>
      </Collapsible>
    </Frame>
  );
}

const HEADER_ICON_BUTTON =
  "inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground";

function useStopRequested(status: SplitRunPhaseStatus, actionBusy: boolean) {
  const [stopping, setStopping] = useState(false);
  const sawBusy = useRef(false);
  useEffect(() => {
    if (status !== "running") {
      sawBusy.current = false;
      setStopping(false);
      return;
    }
    if (actionBusy) {
      sawBusy.current = true;
      return;
    }
    if (sawBusy.current) {
      sawBusy.current = false;
      setStopping(false);
    }
  }, [actionBusy, status]);
  return { active: stopping, request: () => setStopping(true) };
}

function FullLogButton({ onFullLog }: { onFullLog: () => void }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button type="button" aria-label="Full log" onClick={onFullLog} className={HEADER_ICON_BUTTON}>
          <Maximize2 className="size-3.5" aria-hidden />
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">Full log</TooltipContent>
    </Tooltip>
  );
}

/**
 * The stage stream on the whole popup: a sticky bar with the way back and
 * the run page link, then every step open for reading.
 */
function ConsoleFullLog({
  automation,
  phase,
  organizationId,
  runHref,
  onBack,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  organizationId?: string;
  runHref?: string;
  onBack: () => void;
}) {
  const { latest } = automation;
  return (
    <div className="flex min-w-0 flex-col gap-3" data-testid="redesign-console-full-log">
      <div className="sticky -top-4 z-10 -mx-1 flex items-center gap-2 border-b bg-background px-1 py-2">
        <Button size="sm" variant="ghost" className="h-7 px-2 text-[12px]" onClick={onBack}>
          <ArrowLeft className="size-3.5" aria-hidden />
          Back
        </Button>
        <StageStatusGlyph status={latest.status} />
        <span className="min-w-0 truncate text-[13px] font-medium text-foreground">{automation.name}</span>
        <span className={cn(META_TEXT_CLASSNAME, "ml-auto shrink-0 tabular-nums")}>{runMetaLine(latest)}</span>
        {runHref ? (
          <Button size="sm" variant="ghost" className="h-7 shrink-0 px-2 text-[12px]" asChild>
            <Link href={runHref}>
              <Maximize2 className="size-3.5" aria-hidden />
              View run
            </Link>
          </Button>
        ) : null}
      </div>
      {latest.description ? (
        <div className="text-[12.5px] leading-5 text-muted-foreground">
          <MarkdownContent content={latest.description} variant="workspace" />
        </div>
      ) : null}
      <LiveAgentSteps stage={latest} phase={phase} organizationId={organizationId} expandSteps />
      <div className="border-t pt-3">
        <span className={META_TEXT_CLASSNAME}>{runFooterLine(latest)}</span>
      </div>
    </div>
  );
}

function StepOutputCounts({ stage }: { stage: AutomationStage }) {
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

function ArtifactChip({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const kind = artifact.type.replace(/^TYPE_/i, "").toLowerCase();
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
  const url = safeExternalUrl(extractArtifactUrl(data));
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
  const href = safeExternalUrl(extractArtifactUrl(data) ?? branchTreeUrl(data));
  return <SingleOpenChip icon={<GitBranch aria-hidden />} name={name} href={href} />;
}

function LinkArtifactChip({ artifact }: { artifact: AutomationStage["outputs"]["artifacts"][number] }) {
  const data = toArtifactDataRecord(artifact.data);
  const name = extractArtifactTitle(data) ?? extractArtifactName(data) ?? extractArtifactUrl(data) ?? "Link";
  const href = safeExternalUrl(extractArtifactUrl(data));
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

function CheckBadgeRow({ check }: { check: WorkOrderCheckPresentation }) {
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

function AutomationCardBody({
  automation,
  phase,
  organizationId,
  taskDescription,
  runHref,
  onStop,
  onRetry,
  actionBusy,
  onOpen,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  organizationId?: string;
  taskDescription?: string;
  runHref?: string;
  onStop?: () => void;
  onRetry?: () => void;
  actionBusy: boolean;
  onOpen: () => void;
}) {
  const { latest, runs } = automation;
  const pullRequest = latest.outputs.pullRequests[0];
  const hasOutputs = Boolean(pullRequest) || latest.outputs.artifacts.length > 0;
  const showDescription = showDescriptionInBody(latest);
  const creationDescription = latest.id === "backlog" ? taskDescription?.trim() : undefined;
  return (
    <div className="space-y-3">
      {showDescription ? (
        <div className="text-[12.5px] leading-5 text-muted-foreground">
          <MarkdownContent content={latest.description ?? ""} variant="workspace" />
        </div>
      ) : null}
      {creationDescription ? <ClampedMarkdown content={creationDescription} /> : null}
      <LiveAgentSteps stage={latest} phase={phase} organizationId={organizationId} />
      {hasOutputs || latest.checks.length > 0 ? (
        <div className="flex flex-col gap-2" data-testid={`redesign-console-outputs-${automation.id}`}>
          {pullRequest ? (
            <WorkOrderPullRequestInline pullRequest={pullRequest} showTitle className="text-[12px]" />
          ) : null}
          {latest.outputs.artifacts.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2">
              {latest.outputs.artifacts.map((artifact) => (
                <ArtifactChip key={artifact.id} artifact={artifact} />
              ))}
            </div>
          ) : null}
          {latest.checks.length > 0 ? (
            <div className="flex flex-wrap items-center gap-1.5">
              {latest.checks.map((check) => (
                <CheckBadgeRow key={check.id} check={check} />
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t pt-3">
        <span className={cn(META_TEXT_CLASSNAME, "min-w-0")}>{runFooterLine(latest)}</span>
        <div className="ms-auto flex shrink-0 items-center gap-1.5">
          {runs.length > 1 ? (
            <Button size="sm" variant="ghost" className="h-7 gap-1.5 px-2 text-[12px]" onClick={onOpen}>
              <History className="size-3.5" aria-hidden />
              View {runs.length} runs
            </Button>
          ) : runHref ? (
            <Button size="sm" variant="ghost" className="h-7 gap-1.5 px-2 text-[12px]" asChild>
              <Link href={runHref}>
                <Maximize2 className="size-3.5" aria-hidden />
                View run
              </Link>
            </Button>
          ) : latest.appId ? (
            <Button size="sm" variant="ghost" className="h-7 px-2 text-[12px]" onClick={onOpen}>
              Open run
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
    </div>
  );
}

const DESCRIPTION_CLAMP_PX = 320;

/** The task description on the creation card. Long text clamps with Show more. */
function ClampedMarkdown({ content }: { content: string }) {
  const [expanded, setExpanded] = useState(false);
  const [clamped, setClamped] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    setClamped((bodyRef.current?.scrollHeight ?? 0) > DESCRIPTION_CLAMP_PX + 40);
  }, [content]);
  return (
    <div data-testid="redesign-console-task-description">
      <div
        ref={bodyRef}
        className={cn("relative overflow-hidden text-[13px] leading-6", !expanded && clamped && "max-h-80")}
      >
        <MarkdownContent content={content} variant="workspace" />
        {!expanded && clamped ? (
          <div
            className="absolute inset-x-0 bottom-0 h-12 bg-gradient-to-t from-background to-transparent"
            aria-hidden
          />
        ) : null}
      </div>
      {clamped ? (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="mt-1 h-7 px-2 text-[12px] text-muted-foreground"
          onClick={() => setExpanded((current) => !current)}
        >
          {expanded ? "Show less" : "Show more"}
        </Button>
      ) : null}
    </div>
  );
}

function ConsoleRunsDrawer({
  automation,
  open,
  onOpenChange,
  fixture,
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
}: {
  automation: ConsoleAutomation | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  fixture: SplitRunFixture;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
}) {
  const runs = automation?.runs ?? [];
  const totalCost = runs.reduce((sum, run) => sum + parseUsd(run.cost), 0);
  const summary = [
    `${runs.length} ${runs.length === 1 ? "run" : "runs"}`,
    totalCost > 0 ? `$${totalCost.toFixed(2)} total` : "",
    automation?.latest.model,
  ]
    .filter(Boolean)
    .join(" · ");
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex w-full flex-col gap-3 sm:max-w-3xl"
        data-testid="redesign-console-run-drawer"
      >
        <SheetHeader className="pr-8">
          <SheetTitle className="text-[15px]">{automation?.name ?? "Runs"}</SheetTitle>
          <SheetDescription>{automation ? summary : "Select an automation to read its runs."}</SheetDescription>
        </SheetHeader>
        <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-auto">
          {runs.map((run, index) => {
            const phase = fixture.phases.find((entry) => entry.id === run.id);
            const runHref = phase
              ? splitRunPhaseRunHref({ organizationId, factoryKey, orderNumber, lineId, phase })
              : undefined;
            return (
              <DrawerRun
                key={run.id}
                run={run}
                phase={phase}
                organizationId={organizationId}
                runHref={runHref}
                defaultOpen={index === 0}
              />
            );
          })}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function parseUsd(value?: string): number {
  const amount = Number(value?.replace(/[^0-9.]/g, ""));
  return Number.isFinite(amount) ? amount : 0;
}

/**
 * One historical run inside the drawer. The newest run starts open. The
 * row leads with the outcome glyph, then the trigger that caused the run
 * (rendered, so mentions and comment links work), then the revision sha.
 * The body reuses the card's live step list.
 */
function DrawerRun({
  run,
  phase,
  organizationId,
  runHref,
  defaultOpen,
}: {
  run: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
  runHref?: string;
  defaultOpen: boolean;
}) {
  const revision = run.pullRequestActivity?.revision;
  const plainName = plainRunTitle(run.name);
  return (
    <Frame variant="default" spacing="sm" stacked dense className="[--frame-radius:var(--radius-lg)]">
      <Collapsible defaultOpen={defaultOpen} className="group/run">
        <FrameHeader className="flex min-w-0 flex-row items-center gap-2 py-2">
          <CollapsibleTrigger className="shrink-0" aria-label={`Toggle ${plainName}`}>
            <StageStatusGlyph status={run.status} />
          </CollapsibleTrigger>
          <MarkdownContent
            content={run.name}
            variant="workspace"
            openLinksInNewTab
            linkClassName="font-medium text-current !underline !decoration-current underline-offset-2"
            className="min-w-0 truncate text-left text-[13px] font-medium text-foreground [&_p]:m-0 [&_p]:inline"
          />
          {revision ? (
            <span className="shrink-0 font-mono text-[12px] text-muted-foreground">{revision.sha?.slice(0, 7)}</span>
          ) : null}
          <span className={cn(META_TEXT_CLASSNAME, "ml-auto shrink-0 tabular-nums")}>
            {[formatClock(run.startedAt), run.duration, run.cost].filter(Boolean).join(" · ")}
          </span>
          <CollapsibleTrigger className={HEADER_ICON_BUTTON} tabIndex={-1} aria-hidden>
            <ChevronRight
              className="size-4 shrink-0 transition-transform duration-200 group-data-[state=open]/run:rotate-90"
              aria-hidden
            />
          </CollapsibleTrigger>
        </FrameHeader>
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            {run.description ? (
              <div className="text-[12.5px] leading-5 text-muted-foreground">
                <MarkdownContent content={run.description} variant="workspace" />
              </div>
            ) : null}
            <LiveAgentSteps stage={run} phase={phase} organizationId={organizationId} />
            <div className="flex items-center justify-between gap-2 border-t pt-3">
              <span className={META_TEXT_CLASSNAME}>{runFooterLine(run)}</span>
              {runHref ? (
                <Button size="sm" variant="ghost" className="h-7 px-2 text-[12px]" asChild>
                  <Link href={runHref}>
                    <Maximize2 className="size-3.5" aria-hidden />
                    View run
                  </Link>
                </Button>
              ) : null}
            </div>
          </FramePanel>
        </CollapsibleContent>
      </Collapsible>
    </Frame>
  );
}

/** Markdown links reduced to their text, for aria labels. */
function plainRunTitle(name: string): string {
  return name.replace(/\[([^\]]*)\]\([^)]*\)/g, "$1");
}

function ConsoleSummaryPanel({
  fixture,
  outcome,
  stages,
  panelReview,
}: {
  fixture: SplitRunFixture;
  outcome: ReturnType<typeof outcomeSummary>;
  stages: AutomationStage[];
  panelReview?: ReactNode;
}) {
  const artifacts = splitRunLinkedArtifacts([
    ...new Map(stages.flatMap((stage) => stage.outputs.artifacts).map((artifact) => [artifact.id, artifact])).values(),
  ]);
  const checks = stages.flatMap((stage) => stage.checks);
  const hasOutputs = outcome.pullRequests.length > 0 || artifacts.length > 0 || checks.length > 0;
  const spendRows = (fixture.usageByModel ?? []).map((row) => ({
    label: row.model?.split("/").at(-1) ?? row.provider ?? "",
    value: `$${(Number(row.costCents ?? 0) / 100).toFixed(2)}`,
  }));
  return (
    <aside className="lg:sticky lg:top-0 lg:self-start" data-testid="redesign-console-summary">
      <Frame variant="default" spacing="sm" stacked className="[--frame-radius:var(--radius-lg)]">
        <FrameHeader>
          <FrameTitle className="flex items-center gap-2">
            <StageStatusGlyph status={outcome.status} />
            {outcome.statusLabel}
          </FrameTitle>
          {panelReview ? null : <FrameDescription className="text-[12.5px]">{outcome.headline}</FrameDescription>}
        </FrameHeader>
        {panelReview ? <FramePanel className="py-3">{panelReview}</FramePanel> : null}
        <FramePanel className="flex flex-col gap-2 py-3">
          <SummaryRow label="Owner">
            <OrgUserReference display={outcome.owner} size="xs" nameClassName="text-[13px]" />
          </SummaryRow>
          <SummaryRow label="Started">{outcome.startedLabel.replace(/^Started\s+/i, "")}</SummaryRow>
          <SummaryRow label="Duration">{outcome.duration}</SummaryRow>
          <SummaryRow label="Spend">
            {outcome.spend} <span className="text-muted-foreground">· {outcome.tokens}</span>
          </SummaryRow>
          {spendRows.map((row) => (
            <SummaryRow key={row.label} label={row.label} muted>
              {row.value}
            </SummaryRow>
          ))}
        </FramePanel>
        {hasOutputs ? (
          <FramePanel className="flex flex-col gap-2 py-3">
            <span className="text-[12px] font-medium text-muted-foreground">Outputs</span>
            {outcome.pullRequests.map((pullRequest) => (
              <WorkOrderPullRequestInline key={pullRequest.id} pullRequest={pullRequest} showTitle />
            ))}
            {artifacts.map((artifact) => (
              <WorkOrderArtifactInline
                key={artifact.id}
                artifact={{ id: artifact.id, type: artifact.type ?? "", data: toArtifactDataRecord(artifact.data) }}
              />
            ))}
            {checks.length > 0 ? (
              <>
                <span className="mt-1 text-[12px] font-medium text-muted-foreground">Checks</span>
                <SplitRunCheckPills checks={checks} testId="redesign-console-checks" />
              </>
            ) : null}
          </FramePanel>
        ) : null}
      </Frame>
    </aside>
  );
}

function SummaryRow({ label, children, muted = false }: { label: string; children: ReactNode; muted?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3 text-[13px]">
      <span className={cn("shrink-0 text-muted-foreground", muted && "pl-3 text-[12px]")}>{label}</span>
      <span className={cn("min-w-0 truncate text-right text-foreground tabular-nums", muted && "text-[12px]")}>
        {children}
      </span>
    </div>
  );
}

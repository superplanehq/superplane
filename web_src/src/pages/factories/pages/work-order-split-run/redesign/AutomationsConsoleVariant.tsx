import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/reui/alert";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/ui/sheet";
import { ChevronRight, CircleX, ExternalLink, History, RotateCw } from "lucide-react";
import { useState, type ReactNode } from "react";

import { OrgUserReference } from "../../../OrgUserReference";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { SplitRunCheckPills } from "../SplitRunReview";
import type { SplitRunFixture } from "../splitRunMocks";
import { splitRunLinkedArtifacts } from "../splitRunPopupModel";
import { AgentStepList } from "./AgentStepList";
import {
  activeAgentStep,
  activeStepProgress,
  runFooterLine,
  runMetaLine,
  runResultLine,
  showDescriptionInBody,
} from "./consoleCardText";
import { allStages, outcomeSummary, stagesFromFixture, type AutomationStage } from "./automationsViewModel";
import { META_TEXT_CLASSNAME, formatClock } from "./redesignFormat";
import { StageStatusGlyph, ToolKindIcon } from "./redesignShared";

/**
 * Variant B: automation console. Backlog, Implement, Verify, and Done sit
 * on a timeline. Each card is the latest run of one automation, seen
 * through its agent: the header carries status, a one-line outcome, and
 * this run's spend; the body carries what the run produced (passed) or
 * what the agent does now (running); the footer carries start time,
 * model, and the way into the runs drawer. Canvas nodes are not shown on
 * this tab. Columns the task has not reached read "Not started". A sticky
 * Frame holds the task status, spend, checks, and outputs.
 */
export function AutomationsConsoleVariant({ fixture }: { fixture: SplitRunFixture }) {
  const outcome = outcomeSummary(fixture);
  const groups = stagesFromFixture(fixture);
  const stages = allStages(groups);
  const columns = consoleColumns(groups, fixture.footer.run?.appId);
  const [openAutomation, setOpenAutomation] = useState<ConsoleAutomation | null>(null);

  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_300px]" data-testid="redesign-console-variant">
      <section className="flex min-w-0 flex-col gap-4">
        <Timeline value={reachedColumns(columns)} className="pl-1">
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
                      onOpen={() => setOpenAutomation(automation)}
                    />
                  ))
                )}
              </TimelineContent>
            </TimelineItem>
          ))}
        </Timeline>
      </section>
      <ConsoleSummaryPanel fixture={fixture} outcome={outcome} stages={stages} />
      <ConsoleRunsDrawer
        automation={openAutomation}
        open={openAutomation !== null}
        onOpenChange={(open) => !open && setOpenAutomation(null)}
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
 * Only automation runs make cards. Task stages sit in the column named
 * after them. Pull request activity sits in Verify, except the runs of
 * the automation that closed the task: those sit in Done.
 */
function consoleColumns(groups: ReturnType<typeof stagesFromFixture>, closerAppId?: string): ConsoleColumn[] {
  const pullRequestStages = groups.pullRequestGroups.flatMap((group) => group.stages).filter(isAutomationRun);
  const closedBy = (stage: AutomationStage) => Boolean(closerAppId) && stage.appId === closerAppId;
  return CONSOLE_COLUMNS.map((column) => {
    const stages =
      column.id === "verify"
        ? pullRequestStages.filter((stage) => !closedBy(stage))
        : column.id === "done"
          ? pullRequestStages.filter(closedBy)
          : groups.taskStages
              .filter(isAutomationRun)
              .filter((stage) => (column.names as readonly string[]).includes(stage.name));
    return { id: column.id, title: column.title, automations: automationsFromStages(stages) };
  });
}

function isAutomationRun(stage: AutomationStage): boolean {
  return Boolean(stage.appId);
}

/** Timeline steps to fill: through the last column that has a run. */
function reachedColumns(columns: ConsoleColumn[]): number {
  const reached = columns.map((column) => column.automations.length > 0).lastIndexOf(true);
  return reached + 1;
}

function automationsFromStages(stages: AutomationStage[]): ConsoleAutomation[] {
  const byName = new Map<string, AutomationStage[]>();
  for (const stage of stages) {
    const runs = byName.get(stage.componentName) ?? [];
    runs.push(stage);
    byName.set(stage.componentName, runs);
  }
  return [...byName.entries()].map(([componentName, runs]) => {
    const newestFirst = [...runs].sort(
      (left, right) => Date.parse(right.startedAt ?? "") - Date.parse(left.startedAt ?? ""),
    );
    return {
      id: componentName.toLowerCase().replace(/\W+/g, "-"),
      name: componentName,
      latest: newestFirst[0],
      runs: newestFirst,
    };
  });
}

function ConsoleAutomationCard({ automation, onOpen }: { automation: ConsoleAutomation; onOpen: () => void }) {
  const { latest } = automation;
  const finished = latest.status === "passed";
  return (
    <Frame
      variant="default"
      spacing="sm"
      stacked
      dense
      className="[--frame-radius:var(--radius-lg)]"
      data-testid={`redesign-console-automation-${automation.id}`}
    >
      <Collapsible defaultOpen={!finished} className="group/collapsible">
        <CollapsibleTrigger className="flex w-full min-w-0" aria-label={`Toggle ${automation.name} details`}>
          <FrameHeader className="flex min-w-0 grow flex-row items-center gap-2 py-2">
            <StageStatusGlyph status={latest.status} />
            <span className="shrink-0 text-[13px] font-medium text-foreground">{automation.name}</span>
            <span className={cn(META_TEXT_CLASSNAME, "min-w-0 flex-1 truncate text-left")}>
              {runResultLine(latest)}
            </span>
            <span className={cn(META_TEXT_CLASSNAME, "ml-auto shrink-0 tabular-nums")}>{runMetaLine(latest)}</span>
            <ChevronRight
              className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
              aria-hidden
            />
          </FrameHeader>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            <AutomationCardBody automation={automation} onOpen={onOpen} />
          </FramePanel>
        </CollapsibleContent>
      </Collapsible>
    </Frame>
  );
}

function AutomationCardBody({ automation, onOpen }: { automation: ConsoleAutomation; onOpen: () => void }) {
  const { latest, runs } = automation;
  const failed = latest.status === "failed" || latest.status === "cancelled";
  const running = latest.status === "running";
  const pullRequest = latest.outputs.pullRequests[0];
  const hasOutputs = Boolean(pullRequest) || latest.outputs.artifacts.length > 0;
  const showDescription = showDescriptionInBody(latest);
  return (
    <div className="space-y-3">
      {showDescription ? (
        <div className="text-[12.5px] leading-5 text-muted-foreground">
          <MarkdownContent content={latest.description ?? ""} variant="workspace" />
        </div>
      ) : null}
      {running ? <LiveActivity stage={latest} /> : null}
      {hasOutputs ? (
        <div className="flex flex-col gap-1.5" data-testid={`redesign-console-outputs-${automation.id}`}>
          {pullRequest ? (
            <WorkOrderPullRequestInline pullRequest={pullRequest} showTitle className="text-[12px]" />
          ) : null}
          {latest.outputs.artifacts.map((artifact) => (
            <WorkOrderArtifactInline
              key={artifact.id}
              artifact={{ id: artifact.id, type: artifact.type ?? "", data: toArtifactDataRecord(artifact.data) }}
            />
          ))}
        </div>
      ) : null}
      {failed ? (
        <Alert variant="destructive">
          <CircleX />
          <AlertTitle>{automation.name} did not finish</AlertTitle>
          <AlertDescription>Fix the error, then run this automation again.</AlertDescription>
          <AlertAction>
            <Button size="sm" variant="outline">
              <RotateCw className="size-3.5" aria-hidden />
              Retry
            </Button>
          </AlertAction>
        </Alert>
      ) : null}
      <div className="flex flex-wrap items-center justify-between gap-2 border-t pt-3">
        <span className={META_TEXT_CLASSNAME}>{runFooterLine(latest)}</span>
        <Button size="sm" variant="ghost" className="h-7 px-2 text-[12px]" onClick={onOpen}>
          {runs.length === 1 ? (
            "Open run"
          ) : (
            <>
              <History className="size-3.5" aria-hidden />
              View {runs.length} runs
            </>
          )}
        </Button>
      </div>
    </div>
  );
}

const LIVE_TAIL_LENGTH = 3;

/**
 * What the agent does right now: its active step, the last thing it said,
 * and its last few tool calls. Only events that already happened; the
 * canvas branches, so nothing after the current node is known. Canvas
 * nodes are not shown; the canvas run page has those. The full transcript
 * stays in the runs drawer.
 */
function LiveActivity({ stage }: { stage: AutomationStage }) {
  const active = activeAgentStep(stage);
  if (!active) {
    return null;
  }
  const lastNote = [...active.events].reverse().find((event) => event.kind === "note");
  const tools = active.events.flatMap((event) => (event.kind === "tools" ? event.tools : []));
  const tail = tools.slice(-LIVE_TAIL_LENGTH);
  return (
    <div className="flex flex-col gap-1.5" data-testid={`redesign-console-live-${stage.id}`}>
      <div className="flex items-center gap-2">
        <StageStatusGlyph status="running" className="size-3.5" />
        <span className="text-[13px] font-medium text-foreground">{active.title}</span>
        <span className={META_TEXT_CLASSNAME}>{activeStepProgress(active)}</span>
      </div>
      {lastNote?.kind === "note" ? (
        <p className="line-clamp-2 text-[12.5px] leading-5 text-foreground/90">{lastNote.text}</p>
      ) : null}
      {tail.length > 0 ? (
        <ul className="flex flex-col gap-1 border-l border-border/70 pl-3">
          {tail.map((tool) => (
            <li key={tool.id} className="flex min-w-0 items-center gap-2">
              <ToolKindIcon type={tool.type} />
              <code className="min-w-0 flex-1 truncate font-mono text-[12px] text-foreground/90" title={tool.name}>
                {tool.name}
              </code>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function ConsoleRunsDrawer({
  automation,
  open,
  onOpenChange,
}: {
  automation: ConsoleAutomation | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
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
          {runs.map((run, index) => (
            <DrawerRun key={run.id} run={run} defaultOpen={index === 0} />
          ))}
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
 * One historical run inside the drawer. The newest run starts open. A run
 * shows its agent transcript with tool calls expanded. Canvas nodes are
 * not listed; "Open on canvas" leads to the run page for those.
 */
function DrawerRun({ run, defaultOpen }: { run: AutomationStage; defaultOpen: boolean }) {
  const revision = run.pullRequestActivity?.revision;
  return (
    <Frame variant="default" spacing="sm" stacked dense className="[--frame-radius:var(--radius-lg)]">
      <Collapsible defaultOpen={defaultOpen} className="group/run">
        <CollapsibleTrigger className="flex w-full min-w-0" aria-label={`Toggle ${run.name}`}>
          <FrameHeader className="flex min-w-0 grow flex-row items-center gap-2 py-2">
            <StageStatusGlyph status={run.status} />
            <span className="min-w-0 truncate text-left text-[13px] font-medium text-foreground">{run.name}</span>
            {revision ? (
              <span className="shrink-0 font-mono text-[12px] text-muted-foreground">{revision.sha?.slice(0, 7)}</span>
            ) : null}
            <span className={cn(META_TEXT_CLASSNAME, "ml-auto shrink-0 tabular-nums")}>
              {[formatClock(run.startedAt), run.duration, run.cost].filter(Boolean).join(" · ")}
            </span>
            <ChevronRight
              className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]/run:rotate-90"
              aria-hidden
            />
          </FrameHeader>
        </CollapsibleTrigger>
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            {run.description ? (
              <div className="text-[12.5px] leading-5 text-muted-foreground">
                <MarkdownContent content={run.description} variant="workspace" />
              </div>
            ) : null}
            <AgentStepList stage={run} view="detailed" showToggle={false} />
            <div className="flex items-center justify-between gap-2 border-t pt-3">
              <span className={META_TEXT_CLASSNAME}>{runFooterLine(run)}</span>
              <Button size="sm" variant="ghost" className="h-7 px-2 text-[12px]">
                <ExternalLink className="size-3.5" aria-hidden />
                Open on canvas
              </Button>
            </div>
          </FramePanel>
        </CollapsibleContent>
      </Collapsible>
    </Frame>
  );
}

function ConsoleSummaryPanel({
  fixture,
  outcome,
  stages,
}: {
  fixture: SplitRunFixture;
  outcome: ReturnType<typeof outcomeSummary>;
  stages: AutomationStage[];
}) {
  const artifacts = splitRunLinkedArtifacts([
    ...new Map(stages.flatMap((stage) => stage.outputs.artifacts).map((artifact) => [artifact.id, artifact])).values(),
  ]);
  const checks = stages.flatMap((stage) => stage.checks);
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
          <FrameDescription className="text-[12.5px]">{outcome.headline}</FrameDescription>
        </FrameHeader>
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
        <FramePanel className="flex flex-wrap gap-2 py-3">
          <Button size="sm" variant="outline">
            <ExternalLink className="size-3.5" aria-hidden />
            View run
          </Button>
          {fixture.footer.actions.map((action) => (
            <Button key={action.id} size="sm" variant={action.emphasis === "primary" ? "default" : "outline"}>
              {action.label}
            </Button>
          ))}
        </FramePanel>
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

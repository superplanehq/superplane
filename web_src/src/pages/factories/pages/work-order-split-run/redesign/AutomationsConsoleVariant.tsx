import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
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
import { Badge } from "@/components/reui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ArrowLeft, ChevronRight, Maximize2, Minimize2 } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FilesFile } from "@/api-client";

import { type SplitRunFixture, type SplitRunPhase, type SplitRunPhaseStatus } from "../splitRunMocks";
import type { SplitRunSource } from "../splitRunSource";
import { AutomationCardBody } from "./AutomationCardBody";
import { ConsoleRunsDrawer } from "./ConsoleRunsDrawer";
import { ConsoleSummaryPanel } from "./ConsoleSummaryPanel";
import { StepOutputCounts } from "./consoleOutputChips";
import { LiveAgentSteps } from "./LiveAgentSteps";
import { runFooterLine, runMetaLine } from "./consoleCardText";
import {
  allStages,
  automationsFromStages,
  outcomeSummary,
  stagesByConsoleColumn,
  stagesFromFixture,
  type ConsoleAutomation,
} from "./automationsViewModel";
import { HEADER_ICON_BUTTON, META_TEXT_CLASSNAME } from "./redesignFormat";
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
  factoryId,
  orderId,
  taskDescription,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  source,
  files,
  pullRequests,
  panelReview,
  canStopRun = false,
  actionBusy = false,
  onStopRun,
  onRerunStep,
}: {
  fixture: SplitRunFixture;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  /** Task description markdown for the creation card body. */
  taskDescription?: string;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  source?: SplitRunSource;
  files?: FilesFile[];
  /** Pull requests tracked on this task, for the summary panel. */
  pullRequests?: FactoriesFactoryPullRequest[];
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
                  <span className={META_TEXT_CLASSNAME}>{index + 1 < currentColumn ? "Skipped" : "Not started"}</span>
                ) : (
                  column.automations.map((automation) => (
                    <ConsoleAutomationCard
                      key={automation.id}
                      automation={automation}
                      phase={fixture.phases.find((phase) => phase.id === automation.latest.id)}
                      organizationId={organizationId}
                      factoryId={factoryId}
                      orderId={orderId}
                      taskDescription={taskDescription}
                      canEditDescription={canEditDescription}
                      descriptionBusy={descriptionBusy}
                      onDescriptionSave={onDescriptionSave}
                      source={source}
                      files={files}
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
      <ConsoleSummaryPanel
        fixture={fixture}
        outcome={outcome}
        stages={stages}
        pullRequests={pullRequests}
        panelReview={panelReview}
        source={source}
        files={files}
      />
      <ConsoleRunsDrawer
        automation={openAutomation}
        open={openAutomation !== null}
        onOpenChange={(open) => !open && setOpenAutomation(null)}
        fixture={fixture}
        organizationId={organizationId}
      />
    </div>
  );
}

const CONSOLE_COLUMNS = [
  { id: "backlog", title: "Backlog" },
  { id: "implement", title: "Implement" },
  { id: "verify", title: "Verify" },
  { id: "done", title: "Done" },
] as const;

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
  const byColumn = stagesByConsoleColumn(groups, closerAppId);
  return CONSOLE_COLUMNS.map((column) => ({
    id: column.id,
    title: column.title,
    automations: automationsFromStages(byColumn[column.id]),
  }));
}

/** Timeline steps to fill: through the last column that has a run. */
function reachedColumns(columns: ConsoleColumn[]): number {
  const reached = columns.map((column) => column.automations.length > 0).lastIndexOf(true);
  return reached + 1;
}

function ConsoleAutomationCard({
  automation,
  phase,
  organizationId,
  factoryId,
  orderId,
  taskDescription,
  canEditDescription,
  descriptionBusy,
  onDescriptionSave,
  source,
  files,
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
  factoryId?: string;
  orderId?: string;
  taskDescription?: string;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  /** Where the task came from. The Backlog creation card shows it. */
  source?: SplitRunSource;
  files?: FilesFile[];
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
        <FrameHeader
          className="relative flex min-w-0 flex-row items-center gap-2 py-2"
          data-testid={`redesign-console-card-header-${automation.id}`}
        >
          <CollapsibleTrigger
            className="absolute inset-0 z-10 cursor-pointer rounded-[inherit]"
            aria-label={`Toggle ${automation.name} details`}
          />
          <StageStatusGlyph status={shownStatus} />
          <span className="shrink-0 text-[13px] font-medium text-foreground">{automation.name}</span>
          <div className="relative z-20">
            <StepOutputCounts stage={latest} />
          </div>
          <span className={cn(META_TEXT_CLASSNAME, "ml-auto px-1.5 tabular-nums")}>{runMetaLine(latest)}</span>
          {latest.appId ? (
            <div className="relative z-20">
              <FullLogButton onFullLog={onFullLog} />
            </div>
          ) : null}
          <ChevronRight
            className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
            aria-hidden
          />
        </FrameHeader>
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            <AutomationCardBody
              automation={automation}
              phase={shownPhase}
              organizationId={organizationId}
              factoryId={factoryId}
              orderId={orderId}
              taskDescription={taskDescription}
              canEditDescription={canEditDescription}
              descriptionBusy={descriptionBusy}
              onDescriptionSave={onDescriptionSave}
              source={source}
              files={files}
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
 * The stage stream on the whole popup: a sticky bar with the way back,
 * then every step open for reading.
 */
function ConsoleFullLog({
  automation,
  phase,
  organizationId,
  onBack,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  organizationId?: string;
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
        <Tooltip>
          <TooltipTrigger asChild>
            <button type="button" aria-label="Collapse full log" onClick={onBack} className={HEADER_ICON_BUTTON}>
              <Minimize2 className="size-3.5" aria-hidden />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top">Collapse</TooltipContent>
        </Tooltip>
      </div>
      {latest.description ? (
        <div className="text-[12.5px] leading-5 text-muted-foreground">
          <MarkdownContent content={latest.description} variant="workspace" />
        </div>
      ) : null}
      <LiveAgentSteps stage={latest} phase={phase} organizationId={organizationId} expandSteps />
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t pt-3">
        <span className={cn(META_TEXT_CLASSNAME, "min-w-0")}>{runFooterLine(latest)}</span>
      </div>
    </div>
  );
}

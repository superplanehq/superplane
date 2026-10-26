import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { durationLabelMs, formatMinutesSecondsDuration, isSubSecondDurationLabel } from "@/lib/duration";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { type SplitRunPhase, type SplitRunPhaseStatus } from "../splitRunMocks";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { AutomationCardBody } from "./AutomationCardBody";
import {
  consoleCanStop,
  consoleHeaderStatus,
  consoleStopRunId,
  runMetaLine,
  type ConsoleDisplayStatus,
} from "./consoleCardText";
import { StepOutputCounts } from "./consoleOutputChips";
import type { ConsoleAutomation } from "./automationsViewModel";
import { META_TEXT_CLASSNAME } from "./redesignFormat";
import { StageStatusGlyph } from "./redesignShared";

function isLiveAutomationStatus(status: ConsoleDisplayStatus): boolean {
  return status === "running" || status === "waiting" || status === "queued";
}

export function ConsoleAutomationCard({
  automation,
  phase,
  phases,
  organizationId,
  factoryKey,
  orderNumber,
  expandIdle,
  canStopRun,
  actionBusy,
  onStopRun,
  onRerunStep,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  phases: SplitRunPhase[];
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  /** True for the current-column card on a draft that has not started a run. */
  expandIdle: boolean;
  canStopRun: boolean;
  actionBusy: boolean;
  onStopRun?: (run: { appId: string; runId: string }) => void;
  onRerunStep?: (phase: SplitRunPhase) => void;
}) {
  const { latest } = automation;
  const { queuedIds, report } = useAgentQueueReports();
  const headerStatus = consoleHeaderStatus(automation.runs, queuedIds);
  const stopping = useStopRequested(latest.status, actionBusy);
  const shownStatus: ConsoleDisplayStatus = stopping.active && latest.status === "running" ? "cancelled" : headerStatus;
  const phaseStatus = shownPhaseStatus(shownStatus);
  const shownPhase = phase && phaseStatus !== phase.status ? { ...phase, status: phaseStatus } : phase;
  const live = isLiveAutomationStatus(shownStatus);
  const timedRuns = useRunDurationClock(automation.runs, phases, queuedIds);
  const [open, setOpen] = useState(live || expandIdle);
  const closedByUser = useRef(false);
  useEffect(() => {
    if (live) {
      setOpen(true);
      return;
    }
    if (closedByUser.current) {
      setOpen(false);
      return;
    }
    if (expandIdle) {
      return;
    }
    setOpen(false);
  }, [expandIdle, live]);
  const handleOpenChange = (next: boolean) => {
    closedByUser.current = !next;
    setOpen(next);
  };
  const stopPhase = phaseForStop(phases, automation.runs, queuedIds) ?? shownPhase;
  const stopRun =
    canStopRun &&
    onStopRun &&
    consoleCanStop(shownStatus, automation.runs, queuedIds) &&
    stopPhase?.appId &&
    stopPhase.runId
      ? () => {
          stopping.request();
          onStopRun({ appId: stopPhase.appId ?? "", runId: stopPhase.runId ?? "" });
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
      {probedPhases(phases, automation).map((entry) => (
        <AgentQueueProbe key={entry.id} organizationId={organizationId} phase={entry} onQueue={report} />
      ))}
      <Collapsible open={open} onOpenChange={handleOpenChange} className="group/collapsible">
        <ConsoleCardHeader
          automation={automation}
          shownStatus={shownStatus}
          shownPhase={shownPhase}
          timer={shownStatus === "queued" ? "" : runMetaLine(latest, timedRuns, queuedIds)}
          queueSummary={mixedQueueSummary(automation.runs, queuedIds)}
        />
        <CollapsibleContent>
          <FramePanel className="space-y-3">
            <AutomationCardBody
              queued={shownStatus === "queued"}
              automation={automation}
              phase={shownPhase}
              phases={phases.map((entry) => (entry.id === latest.id && shownPhase ? shownPhase : entry))}
              organizationId={organizationId}
              factoryKey={factoryKey}
              orderNumber={orderNumber}
              onStop={stopRun}
              onRetry={rerunStep}
              actionBusy={actionBusy}
            />
          </FramePanel>
        </CollapsibleContent>
      </Collapsible>
    </Frame>
  );
}

function ConsoleCardHeader({
  automation,
  shownStatus,
  shownPhase,
  timer,
  queueSummary,
}: {
  automation: ConsoleAutomation;
  shownStatus: ConsoleDisplayStatus;
  shownPhase?: SplitRunPhase;
  timer: string;
  queueSummary: string;
}) {
  return (
    <FrameHeader
      className="relative flex min-w-0 flex-row items-center gap-2 py-2"
      data-testid={`redesign-console-card-header-${automation.id}`}
    >
      <CollapsibleTrigger
        className="absolute inset-0 z-10 cursor-pointer rounded-[inherit]"
        aria-label={`Toggle ${automation.name} details`}
      />
      <StageStatusGlyph status={shownStatus} />
      {shownStatus === "cancelled" ? (
        <span className="shrink-0 text-[13px] font-medium text-muted-foreground">Stopped</span>
      ) : null}
      {shownStatus === "queued" ? (
        <span className="shrink-0 text-[13px] font-medium text-muted-foreground">Queued</span>
      ) : null}
      <span className="shrink-0 text-[13px] font-medium text-foreground">{automation.name}</span>
      {queueSummary ? (
        <span className={META_TEXT_CLASSNAME}>{queueSummary}</span>
      ) : (
        <StepOutputCounts stage={automation.latest} phase={shownPhase} runs={automation.runs} />
      )}
      {timer ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              tabIndex={0}
              className={cn(META_TEXT_CLASSNAME, "relative z-20 ml-auto cursor-help px-1.5 tabular-nums")}
            >
              Run time {timer}
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-64">
            Sum of run durations, including setup and waits. Concurrent runs count separately. Runs currently queued do
            not add time.
          </TooltipContent>
        </Tooltip>
      ) : (
        <span className="ml-auto" />
      )}
      <ChevronRight
        className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
        aria-hidden
      />
    </FrameHeader>
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

function shownPhaseStatus(status: ConsoleDisplayStatus): SplitRunPhaseStatus {
  return status === "queued" ? "running" : status;
}

function probedPhases(phases: SplitRunPhase[], automation: ConsoleAutomation): SplitRunPhase[] {
  const runIds = new Set(automation.runs.map((run) => run.id));
  return phases.filter((phase) => runIds.has(phase.id) && phase.appId && phase.runId && phase.status === "running");
}

function phaseForStop(
  phases: SplitRunPhase[],
  runs: ConsoleAutomation["runs"],
  queuedIds: ReadonlySet<string>,
): SplitRunPhase | undefined {
  const runId = consoleStopRunId(runs, queuedIds);
  return runId ? phases.find((phase) => phase.id === runId) : undefined;
}

function useAgentQueueReports() {
  const [queuedIds, setQueuedIds] = useState<ReadonlySet<string>>(() => new Set());
  const report = useCallback((id: string, queued: boolean) => {
    setQueuedIds((current) => {
      if (current.has(id) === queued) {
        return current;
      }
      const copy = new Set(current);
      if (queued) {
        copy.add(id);
      } else {
        copy.delete(id);
      }
      return copy;
    });
  }, []);
  return { queuedIds, report };
}

function AgentQueueProbe({
  organizationId,
  phase,
  onQueue,
}: {
  organizationId?: string;
  phase: SplitRunPhase;
  onQueue: (id: string, queued: boolean) => void;
}) {
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const queued = Boolean(live.agentQueue?.queued);
  useEffect(() => {
    if (!live.agentQueue) {
      return;
    }
    onQueue(phase.id, queued);
  }, [live.agentQueue, onQueue, phase.id, queued]);
  return null;
}

type RunDurationSample = { duration: string; runId?: string; running: boolean; at: number };

function useRunDurationClock(
  runs: ConsoleAutomation["runs"],
  phases: SplitRunPhase[],
  queuedIds: ReadonlySet<string>,
): ConsoleAutomation["runs"] {
  const samples = useRef(new Map<string, RunDurationSample>());
  const phaseById = new Map(phases.map((phase) => [phase.id, phase]));
  const isRunning = (run: ConsoleAutomation["latest"]) =>
    phaseById.get(run.id)?.durationRunning ?? run.status === "running";
  const now = useTickingNow(runs.some((run) => isRunning(run) && !queuedIds.has(run.id)));
  return runs.map((run) => {
    const running = isRunning(run);
    const runId = phaseById.get(run.id)?.runId;
    let sample = samples.current.get(run.id);
    if (!sample || sample.duration !== run.duration || sample.runId !== runId || sample.running !== running) {
      sample = { duration: run.duration, runId, running, at: Date.now() };
      samples.current.set(run.id, sample);
    }
    const durationMs = durationLabelMs(run.duration);
    if (!running || (durationMs === 0 && !isSubSecondDurationLabel(run.duration))) {
      return run;
    }
    // Analysis can combine attempts. Advance its sampled total, not the time
    // since the first attempt, which can include gaps between attempts.
    const elapsedMs = Math.max(0, Math.floor((now - sample.at) / 1000) * 1000);
    return { ...run, duration: formatMinutesSecondsDuration(durationMs + elapsedMs) || "<1s" };
  });
}

function useTickingNow(active: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) {
      return;
    }
    setNow(Date.now());
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [active]);
  return now;
}

function mixedQueueSummary(runs: ConsoleAutomation["runs"], queuedIds: ReadonlySet<string>): string {
  const active = runs.filter((run) => run.status === "running");
  const queued = active.filter((run) => queuedIds.has(run.id)).length;
  const running = active.length - queued;
  return running > 0 && queued > 0 ? `${running} running · ${queued} queued` : "";
}

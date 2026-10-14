import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { cn } from "@/lib/utils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { type SplitRunPhase, type SplitRunPhaseStatus } from "../splitRunMocks";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { AutomationCardBody } from "./AutomationCardBody";
import {
  consoleCanStop,
  consoleHeaderStatus,
  consoleStopRunId,
  runMetaLine,
  type ConsoleDisplayStatus,
  type ConsoleRunTiming,
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
  const { queuedIds, timing, report } = useAgentQueueReports();
  const headerStatus = consoleHeaderStatus(automation.runs, queuedIds);
  const stopping = useStopRequested(latest.status, actionBusy);
  const shownStatus: ConsoleDisplayStatus = stopping.active && latest.status === "running" ? "cancelled" : headerStatus;
  const phaseStatus = shownPhaseStatus(shownStatus);
  const shownPhase = phase && phaseStatus !== phase.status ? { ...phase, status: phaseStatus } : phase;
  const live = isLiveAutomationStatus(shownStatus);
  const ticking =
    headerStatus === "running" && [...timing.values()].some((entry) => entry.workStartedAt && !entry.workEndedAt);
  const now = useTickingNow(ticking);
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
      {probedPhases(phases, automation, timing).map((entry) => (
        <AgentQueueProbe key={entry.id} organizationId={organizationId} phase={entry} onQueue={report} />
      ))}
      <Collapsible open={open} onOpenChange={handleOpenChange} className="group/collapsible">
        <ConsoleCardHeader
          automation={automation}
          shownStatus={shownStatus}
          shownPhase={shownPhase}
          timer={shownStatus === "queued" ? "" : runMetaLine(latest, automation.runs, timing, now)}
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
      <span className={cn(META_TEXT_CLASSNAME, "ml-auto px-1.5 tabular-nums")}>{timer}</span>
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

function probedPhases(
  phases: SplitRunPhase[],
  automation: ConsoleAutomation,
  timing: ReadonlyMap<string, ConsoleRunTiming>,
): SplitRunPhase[] {
  const runIds = new Set(automation.runs.map((run) => run.id));
  return phases.filter((phase) => {
    if (!runIds.has(phase.id) || !phase.appId || !phase.runId) {
      return false;
    }
    if (phase.status === "running") {
      return true;
    }
    const known = timing.get(phase.id);
    return Boolean(known?.workStartedAt && !known.workEndedAt);
  });
}

function phaseForStop(
  phases: SplitRunPhase[],
  runs: ConsoleAutomation["runs"],
  queuedIds: ReadonlySet<string>,
): SplitRunPhase | undefined {
  const runId = consoleStopRunId(runs, queuedIds);
  return runId ? phases.find((phase) => phase.id === runId) : undefined;
}

function sameOpenStarts(previous: string[] | undefined, next: string[] | undefined): boolean {
  if (previous === next) {
    return true;
  }
  if (!previous || !next || previous.length !== next.length) {
    return false;
  }
  return previous.every((value, index) => value === next[index]);
}

function useAgentQueueReports() {
  const [timing, setTiming] = useState<ReadonlyMap<string, ConsoleRunTiming>>(() => new Map());
  const report = useCallback((id: string, next: ConsoleRunTiming) => {
    setTiming((current) => {
      const previous = current.get(id);
      if (
        previous?.queued === next.queued &&
        previous?.workStartedAt === next.workStartedAt &&
        previous?.workEndedAt === next.workEndedAt &&
        previous?.completedWorkMs === next.completedWorkMs &&
        sameOpenStarts(previous?.openStartedAt, next.openStartedAt)
      ) {
        return current;
      }
      const copy = new Map(current);
      copy.set(id, next);
      return copy;
    });
  }, []);
  const queuedIds = useMemo(() => {
    const ids = new Set<string>();
    for (const [id, entry] of timing) {
      if (entry.queued) {
        ids.add(id);
      }
    }
    return ids;
  }, [timing]);
  return { queuedIds, timing, report };
}

function AgentQueueProbe({
  organizationId,
  phase,
  onQueue,
}: {
  organizationId?: string;
  phase: SplitRunPhase;
  onQueue: (id: string, timing: ConsoleRunTiming) => void;
}) {
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const queued = Boolean(live.agentQueue?.queued);
  const workStartedAt = live.agentQueue?.workStartedAt;
  const workEndedAt = live.agentQueue?.workEndedAt;
  const completedWorkMs = live.agentQueue?.completedWorkMs;
  const openStartedAt = live.agentQueue?.openStartedAt;
  useEffect(() => {
    if (!live.agentQueue) {
      return;
    }
    onQueue(phase.id, { queued, workStartedAt, workEndedAt, completedWorkMs, openStartedAt });
  }, [completedWorkMs, live.agentQueue, onQueue, openStartedAt, phase.id, queued, workEndedAt, workStartedAt]);
  return null;
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

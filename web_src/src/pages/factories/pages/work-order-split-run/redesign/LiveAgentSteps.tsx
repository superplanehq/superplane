import type { AgentPromptUsageSeries } from "@/lib/agentRunTelemetry";
import { cn } from "@/lib/utils";
import { useRevealAfterPending } from "@/hooks/useRevealAfterPending";
import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { Skeleton } from "@/ui/skeleton";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { LOADING_REVEAL_CLASSNAME } from "../../../lib/loadingReveal";
import { mergeAgentActivities, type AgentActivity } from "../agentActivity";
import { AgentLiveStatus } from "../IntentAnalysisLiveWork";
import { AgentActivityView } from "../AgentActivityView";
import { headerSpendFromUsageSeries } from "../planningHeaderSpend";
import { useReportLiveHeaderSpend } from "../liveHeaderSpendContext";
import { useReportPhaseAgentUsageSeries } from "../phaseAgentUsageContext";
import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "../splitRunMocks";
import { activitiesFromLiveLogSections, isRunnerComponent, notesForLiveStream } from "../streamNotesFromLiveLog";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { activityFromTranscript } from "./activityFromAgentStep";
import { AgentStepMarkers } from "./AgentStepList";
import { agentStepsFromNotes, settleStoppedSteps, type AutomationStage } from "./automationsViewModel";
import { FailedNodeAlerts } from "./failedNodeAlerts";
import { failedNonRunnerErrors, type FailedNodeError } from "./failedNodeErrors";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

const WAITING_FOR_LOGS_NOTE = "Waiting for logs…";
const AGENT_LOG_SKELETON_LABEL = "Waiting for logs";
const AGENT_LOG_SKELETON_WIDTHS = ["w-full", "w-11/12", "w-4/5", "w-2/3"] as const;

type RunnerLive = {
  notes: SplitRunStreamLine[];
  activities: AgentActivity[];
  isStreaming: boolean;
  isLoading: boolean;
};

export function LiveAgentSteps({
  stage,
  phase,
  organizationId,
  expandSteps = false,
  emptyNote,
  reportUsage = true,
  runHref,
}: {
  stage: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
  /** Open every step's detail, for the full-log view. */
  expandSteps?: boolean;
  /** Shown when a finished run has no transcript to replay. */
  emptyNote?: string;
  /** When false, skip the usage chart. The card footer only charts the latest run. */
  reportUsage?: boolean;
  /** Canvas run page for a failed node that is not a runner. */
  runHref?: string | null;
}) {
  const run = useLiveAgentRun(stage, phase, organizationId);
  const note = emptyNote && stage.status !== "running" ? <p className={META_TEXT_CLASSNAME}>{emptyNote}</p> : null;
  if (!organizationId || !phase) {
    return (
      <SettledRunBody
        stage={stage}
        expandSteps={expandSteps}
        showLogSkeleton={run.showLogSkeleton}
        stoppedStatus={run.stoppedStatus}
        note={note}
      />
    );
  }
  return (
    <>
      {run.runners.map((line) => (
        <RunnerNotes
          key={line.id}
          line={line}
          organizationId={organizationId}
          canvasId={phase.appId ?? ""}
          spendPhaseId={phase.id}
          reportUsage={reportUsage}
          onLive={run.reportLive}
        />
      ))}
      <AgentRunBody
        stageId={stage.id}
        shown={run.shown}
        showLogSkeleton={run.showLogSkeleton}
        revealSteps={run.revealSteps}
        expandSteps={expandSteps}
        running={run.running}
        liveActivity={run.liveActivity}
        hasLiveContent={run.hasLiveContent}
        note={note}
        runHref={runHref}
        failedNodes={run.failedNodes}
      />
    </>
  );
}

function SettledRunBody({
  stage,
  expandSteps,
  showLogSkeleton,
  stoppedStatus,
  note,
}: {
  stage: AutomationStage;
  expandSteps: boolean;
  showLogSkeleton: boolean;
  stoppedStatus: SplitRunPhaseStatus | undefined;
  note: ReactNode;
}) {
  if (showLogSkeleton) {
    return <AgentLogSkeleton />;
  }
  const settled = { ...stage, agentSteps: settleStoppedSteps(stage.agentSteps, stoppedStatus) };
  return settled.agentSteps.length > 0 ? <AgentStepMarkers stage={settled} expandSteps={expandSteps} /> : note;
}

function useLiveAgentRun(stage: AutomationStage, phase: SplitRunPhase | undefined, organizationId?: string) {
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const stream = useMemo(
    () => (live.stream.length > 0 ? live.stream : (phase?.stream ?? [])),
    [live.stream, phase?.stream],
  );
  const runners = stream.filter((line) => isRunnerComponent(line.component) && Boolean(line.executionId));
  const [liveByLine, setLiveByLine] = useState<Record<string, RunnerLive>>({});
  const reportLive = useCallback((lineId: string, next: RunnerLive) => {
    setLiveByLine((current) => (sameRunnerLive(current[lineId], next) ? current : { ...current, [lineId]: next }));
  }, []);
  const liveNotes = useMemo(() => runners.flatMap((line) => liveByLine[line.id]?.notes ?? []), [liveByLine, runners]);
  const transcriptNotes = useMemo(() => liveNotes.filter((note) => !isWaitingForLogsNote(note)), [liveNotes]);
  const stoppedStatus = stoppedStepStatus(phase?.status, runners);
  const runStatus = runStatusForAgentSteps(phase?.status, runners);
  const liveSteps = useMemo(
    () => settleStoppedSteps(agentStepsFromNotes(transcriptNotes, runStatus), stoppedStatus),
    [runStatus, transcriptNotes, stoppedStatus],
  );
  const failedNodes = useMemo(() => failedNonRunnerErrors(stream), [stream]);
  const runningRunner = [...runners].reverse().find((line) => line.status === "running" && Boolean(line.executionId));
  const running = Boolean(runningRunner);
  const liveActivity = activityFromTranscript(runningRunner ? (liveByLine[runningRunner.id]?.activities ?? []) : []);
  const hasLiveContent = (liveActivity?.items.length ?? 0) > 0;
  const pending = agentLogPending({
    running,
    canvasLoading: live.isLoading,
    canAwaitNotes: Boolean(phase?.appId),
    runners,
    liveByLine,
    liveNotes,
    liveStepCount: liveSteps.length,
    settledStepCount: stage.agentSteps.length,
    hasLiveContent,
  });
  return {
    runners,
    reportLive,
    running,
    liveActivity,
    hasLiveContent,
    showLogSkeleton: pending.showLogSkeleton,
    revealSteps: useRevealAfterPending(pending.showLogSkeleton),
    shown: {
      ...stage,
      agentSteps: agentStepsForLiveRun(stage.agentSteps, liveSteps, pending.streamWaiting, stoppedStatus),
    },
    failedNodes,
    stoppedStatus,
  };
}

function sameRunnerLive(previous: RunnerLive | undefined, next: RunnerLive): boolean {
  return Boolean(
    previous &&
      previous.isStreaming === next.isStreaming &&
      previous.isLoading === next.isLoading &&
      noteSignature(previous.notes) === noteSignature(next.notes) &&
      activitySignature(previous.activities) === activitySignature(next.activities),
  );
}

function agentLogPending(input: {
  running: boolean;
  canvasLoading: boolean;
  canAwaitNotes: boolean;
  runners: SplitRunStreamLine[];
  liveByLine: Record<string, RunnerLive>;
  liveNotes: SplitRunStreamLine[];
  liveStepCount: number;
  settledStepCount: number;
  hasLiveContent: boolean;
}): { streamWaiting: boolean; showLogSkeleton: boolean } {
  const awaiting = input.canAwaitNotes && input.runners.some((line) => input.liveByLine[line.id] === undefined);
  const loadingLogs = input.runners.some((line) => input.liveByLine[line.id]?.isLoading);
  const pendingFetch = input.canvasLoading || awaiting || loadingLogs;
  const pendingLogs = pendingFetch || isWaitingForLogNotes(input.liveNotes);
  const hasSteps = input.liveStepCount > 0 || input.settledStepCount > 0;
  if (input.hasLiveContent) {
    return { streamWaiting: false, showLogSkeleton: false };
  }
  if (input.running) {
    const wait = pendingLogs && input.liveStepCount === 0;
    return { streamWaiting: wait, showLogSkeleton: wait };
  }
  // A finished run can leave its log SSE open with no bytes. Do not wait
  // on isStreaming. Show steps as soon as any runner has them.
  return { streamWaiting: false, showLogSkeleton: pendingFetch && !hasSteps };
}

function AgentRunBody({
  stageId,
  shown,
  showLogSkeleton,
  revealSteps,
  expandSteps,
  running,
  liveActivity,
  hasLiveContent,
  note,
  runHref,
  failedNodes,
}: {
  stageId: string;
  shown: AutomationStage;
  showLogSkeleton: boolean;
  revealSteps: boolean;
  expandSteps: boolean;
  running: boolean;
  liveActivity?: AgentActivity;
  hasLiveContent: boolean;
  note: ReactNode;
  runHref?: string | null;
  failedNodes: FailedNodeError[];
}) {
  if (showLogSkeleton) {
    return <AgentLogSkeleton />;
  }
  const steps =
    shown.agentSteps.length > 0 ? (
      <div className={cn(revealSteps && LOADING_REVEAL_CLASSNAME)}>
        <AgentStepMarkers
          stage={shown}
          expandSteps={expandSteps}
          liveActivity={running ? liveActivity : undefined}
          liveActive={running}
        />
      </div>
    ) : null;
  const errors = failedNodes.length > 0 ? <FailedNodeAlerts nodes={failedNodes} runHref={runHref} /> : null;
  const liveView = running ? (
    <div data-testid={`redesign-live-activity-${stageId}`}>
      {hasLiveContent && liveActivity ? (
        <AgentActivityView
          activity={liveActivity}
          live
          collapseCompleted={false}
          collapseReasoning={false}
          expandableCommands
          tone="log"
        />
      ) : null}
      <AgentLiveStatus active activity={liveActivity} startingLabel="Starting agent…" collapseReasoning={false} />
    </div>
  ) : null;
  if (steps || errors) {
    return (
      <div className="space-y-3">
        {steps}
        {errors}
        {steps ? null : liveView}
      </div>
    );
  }
  if (!running) {
    return note;
  }
  return liveView;
}

function RunnerNotes({
  line,
  organizationId,
  canvasId,
  spendPhaseId,
  reportUsage,
  onLive,
}: {
  line: SplitRunStreamLine;
  organizationId: string;
  canvasId: string;
  spendPhaseId: string;
  reportUsage: boolean;
  onLive: (lineId: string, live: RunnerLive) => void;
}) {
  const { notes, activities, spend, usageSeries, isStreaming, isLoading } = useRunnerLiveNotes(
    line,
    organizationId,
    canvasId,
  );
  useReportLiveHeaderSpend(`${spendPhaseId}:${line.id}`, spend.tokens, spend.cents);
  useReportPhaseAgentUsageSeries({
    nodeId: line.nodeId ?? line.id,
    fallbackName: line.componentName,
    series: usageSeries,
    enabled: reportUsage && isRunnerComponent(line.component),
  });
  useEffect(() => {
    onLive(line.id, { notes, activities, isStreaming, isLoading });
  }, [activities, isLoading, isStreaming, line.id, notes, onLive]);
  return null;
}

function useRunnerLiveNotes(
  line: SplitRunStreamLine,
  organizationId: string,
  canvasId: string,
): {
  notes: SplitRunStreamLine[];
  activities: AgentActivity[];
  spend: ReturnType<typeof headerSpendFromUsageSeries>;
  usageSeries: AgentPromptUsageSeries[];
  isStreaming: boolean;
  isLoading: boolean;
} {
  const canStream = Boolean(canvasId && line.executionId && isRunnerComponent(line.component));
  const {
    sections,
    orphanLines,
    error,
    isStreaming: streamOpen,
    isLoading: streamLoading,
    usageSeries,
    activityState,
  } = useLiveLogStream(
    canStream ? (line.executionId ?? "") : "",
    line.status === "running",
    liveLogFinishState(line.status),
    null,
    { organizationId, canvasId },
  );
  const isStreaming = canStream && streamOpen;
  const isLoading = canStream && streamLoading;
  const notes = useMemo(() => {
    if (!canStream) {
      return [];
    }
    return (
      notesForLiveStream({
        nodeId: line.nodeId ?? line.id,
        sections,
        orphanLines,
        error,
        isStreaming,
        nodeStatus: line.status,
      }) ?? []
    );
  }, [canStream, error, isStreaming, line.id, line.nodeId, line.status, orphanLines, sections]);
  const activities = useMemo(() => {
    const recorded = mergeAgentActivities(
      sections.flatMap((section) => section.activities ?? []),
      activityState?.activities ?? [],
    );
    if (recorded.some((activity) => activity.items.length > 0)) {
      return recorded;
    }
    return activitiesFromLiveLogSections(sections);
  }, [activityState?.activities, sections]);
  return {
    notes,
    activities,
    spend: headerSpendFromUsageSeries(usageSeries ?? []),
    usageSeries: usageSeries ?? [],
    isStreaming,
    isLoading,
  };
}

function AgentLogSkeleton() {
  return (
    <div className="flex w-full flex-col gap-2" role="status" aria-busy="true" aria-label={AGENT_LOG_SKELETON_LABEL}>
      {AGENT_LOG_SKELETON_WIDTHS.map((width) => (
        <Skeleton key={width} className={cn("h-4", width)} />
      ))}
    </div>
  );
}

function agentStepsForLiveRun(
  settledSteps: AutomationStage["agentSteps"],
  liveSteps: AutomationStage["agentSteps"],
  streamWaiting: boolean,
  stoppedStatus: SplitRunPhaseStatus | undefined,
) {
  if (liveSteps.length > 0) {
    return liveSteps;
  }
  if (streamWaiting) {
    return [];
  }
  return settleStoppedSteps(settledSteps, stoppedStatus);
}

function isWaitingForLogsNote(note: SplitRunStreamLine): boolean {
  return note.componentName === WAITING_FOR_LOGS_NOTE;
}

function isWaitingForLogNotes(notes: SplitRunStreamLine[]): boolean {
  return notes.length > 0 && notes.every(isWaitingForLogsNote);
}

function runStatusForAgentSteps(
  phaseStatus: SplitRunPhaseStatus | undefined,
  runners: SplitRunStreamLine[],
): SplitRunPhaseStatus | undefined {
  if (phaseStatus) {
    return phaseStatus;
  }
  if (runners.some((line) => line.status === "failed")) {
    return "failed";
  }
  if (runners.length > 0 && runners.every((line) => line.status === "passed")) {
    return "passed";
  }
  return undefined;
}

function stoppedStepStatus(
  phaseStatus: SplitRunPhaseStatus | undefined,
  lines: SplitRunStreamLine[],
): SplitRunPhaseStatus | undefined {
  if (phaseStatus === "cancelled" || phaseStatus === "failed") {
    return phaseStatus;
  }
  if (lines.some((line) => line.status === "cancelled")) {
    return "cancelled";
  }
  if (lines.some((line) => line.status === "failed")) {
    return "failed";
  }
  return phaseStatus;
}

function liveLogFinishState(status: SplitRunPhaseStatus): "failed" | "passed" | null {
  if (status === "failed") {
    return "failed";
  }
  if (status === "passed") {
    return "passed";
  }
  return null;
}

function noteSignature(notes: SplitRunStreamLine[]): string {
  return notes.map((note) => `${note.id}\0${note.status}\0${note.componentName}\0${note.detail ?? ""}`).join("\n");
}

function activitySignature(activities: AgentActivity[]): string {
  return activities
    .map((activity) => {
      const last = activity.items.at(-1);
      return `${activity.id}:${activity.sequence}:${activity.status}:${activity.items.length}:${last?.id ?? ""}:${last && "status" in last ? last.status : ""}`;
    })
    .join("|");
}

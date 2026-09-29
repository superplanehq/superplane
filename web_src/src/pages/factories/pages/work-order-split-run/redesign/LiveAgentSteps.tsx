import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { useCallback, useEffect, useMemo, useState } from "react";

import { mergeAgentActivities, type AgentActivity } from "../agentActivity";
import { AgentLiveStatus } from "../IntentAnalysisLiveWork";
import { headerSpendFromUsageSeries } from "../planningHeaderSpend";
import { useReportLiveHeaderSpend } from "../liveHeaderSpendContext";
import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "../splitRunMocks";
import { activitiesFromLiveLogSections, isRunnerComponent, notesForLiveStream } from "../streamNotesFromLiveLog";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { AgentStepMarkers } from "./AgentStepList";
import { agentStepsFromNotes, settleStoppedSteps, type AutomationStage } from "./automationsViewModel";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

type RunnerLive = { notes: SplitRunStreamLine[]; activities: AgentActivity[] };

export function LiveAgentSteps({
  stage,
  phase,
  organizationId,
  expandSteps = false,
  emptyNote,
}: {
  stage: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
  /** Open every step's detail, for the full-log view. */
  expandSteps?: boolean;
  /** Shown when a finished run has no transcript to replay. */
  emptyNote?: string;
}) {
  const live = useSplitRunLiveCanvas(organizationId, phase);
  const stream = live.stream.length > 0 ? live.stream : (phase?.stream ?? []);
  const runners = stream.filter((line) => isRunnerComponent(line.component) && Boolean(line.executionId));
  const [liveByLine, setLiveByLine] = useState<Record<string, RunnerLive>>({});
  const reportLive = useCallback((lineId: string, next: RunnerLive) => {
    setLiveByLine((current) => {
      const previous = current[lineId];
      if (
        previous &&
        noteSignature(previous.notes) === noteSignature(next.notes) &&
        activitySignature(previous.activities) === activitySignature(next.activities)
      ) {
        return current;
      }
      return { ...current, [lineId]: next };
    });
  }, []);
  const liveNotes = useMemo(() => runners.flatMap((line) => liveByLine[line.id]?.notes ?? []), [liveByLine, runners]);
  const stoppedStatus = stoppedStepStatus(phase?.status, runners);
  const liveSteps = useMemo(
    () => settleStoppedSteps(agentStepsFromNotes(liveNotes), stoppedStatus),
    [liveNotes, stoppedStatus],
  );
  const shown =
    liveSteps.length > 0
      ? { ...stage, agentSteps: liveSteps }
      : { ...stage, agentSteps: settleStoppedSteps(stage.agentSteps, stoppedStatus) };

  // A running run shows nothing until its first note streams in; the
  // note is for finished runs whose transcript never arrives.
  const note = emptyNote && stage.status !== "running" ? <p className={META_TEXT_CLASSNAME}>{emptyNote}</p> : null;
  if (!organizationId || !phase || runners.length === 0) {
    const settled = { ...stage, agentSteps: settleStoppedSteps(stage.agentSteps, stoppedStatus) };
    return settled.agentSteps.length > 0 ? <AgentStepMarkers stage={settled} expandSteps={expandSteps} /> : note;
  }

  const running = runners.some((line) => line.status === "running" && Boolean(line.executionId));
  const transcript = runners.flatMap((line) => liveByLine[line.id]?.activities ?? []);
  const lastActivity = [...transcript].reverse().find((activity) => activity.items.length > 0) ?? transcript.at(-1);

  return (
    <>
      {runners.map((line) => (
        <RunnerNotes
          key={line.id}
          line={line}
          organizationId={organizationId}
          canvasId={phase.appId ?? ""}
          spendPhaseId={phase.id}
          onLive={reportLive}
        />
      ))}
      {shown.agentSteps.length > 0 ? (
        <AgentStepMarkers
          stage={shown}
          expandSteps={expandSteps}
          liveActivity={running ? lastActivity : undefined}
          liveActive={running}
        />
      ) : running ? (
        <div data-testid={`redesign-live-activity-${stage.id}`}>
          <AgentLiveStatus active activity={lastActivity} startingLabel="Starting agent…" />
        </div>
      ) : (
        note
      )}
    </>
  );
}

function RunnerNotes({
  line,
  organizationId,
  canvasId,
  spendPhaseId,
  onLive,
}: {
  line: SplitRunStreamLine;
  organizationId: string;
  canvasId: string;
  spendPhaseId: string;
  onLive: (lineId: string, live: RunnerLive) => void;
}) {
  const { notes, activities, spend } = useRunnerLiveNotes(line, organizationId, canvasId);
  useReportLiveHeaderSpend(`${spendPhaseId}:${line.id}`, spend.tokens, spend.cents);
  useEffect(() => {
    onLive(line.id, { notes, activities });
  }, [activities, line.id, notes, onLive]);
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
} {
  const canStream = Boolean(canvasId && line.executionId && isRunnerComponent(line.component));
  const { sections, orphanLines, error, isStreaming, usageSeries, activityState } = useLiveLogStream(
    canStream ? (line.executionId ?? "") : "",
    line.status === "running",
    liveLogFinishState(line.status),
    null,
    { organizationId, canvasId },
  );
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
  return { notes, activities, spend: headerSpendFromUsageSeries(usageSeries ?? []) };
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

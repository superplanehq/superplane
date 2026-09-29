import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";

import { headerSpendFromUsageSeries } from "../planningHeaderSpend";
import { AgentLiveActivity } from "../IntentAnalysisLiveWork";
import { useReportLiveHeaderSpend } from "../liveHeaderSpendContext";
import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "../splitRunMocks";
import { isRunnerComponent, notesForLiveStream } from "../streamNotesFromLiveLog";
import { useFollowLogScroll } from "../useFollowLogScroll";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { JumpToLatestPill } from "../JumpToLatestPill";
import { AgentStepMarkers } from "./AgentStepList";
import { agentStepsFromNotes, settleStoppedSteps, type AutomationStage } from "./automationsViewModel";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

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
  const [notesByLine, setNotesByLine] = useState<Record<string, SplitRunStreamLine[]>>({});
  const reportNotes = useCallback((lineId: string, notes: SplitRunStreamLine[]) => {
    setNotesByLine((current) => {
      const previous = current[lineId];
      if (previous && noteSignature(previous) === noteSignature(notes)) {
        return current;
      }
      return { ...current, [lineId]: notes };
    });
  }, []);
  const liveNotes = useMemo(() => runners.flatMap((line) => notesByLine[line.id] ?? []), [notesByLine, runners]);
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

  // The running run streams like the refinement chat: settled steps stay
  // rows, the running step keeps its title row, and the live activity
  // renders reasoning and tool calls under it as they happen.
  const liveRunner = phase.appId ? runners.find((line) => line.status === "running" && line.executionId) : undefined;
  const steps = liveRunner
    ? shown.agentSteps.map((step) => (step.status === "running" ? { ...step, events: [], output: undefined } : step))
    : shown.agentSteps;

  return (
    <>
      {runners.map((line) => (
        <RunnerNotes
          key={line.id}
          line={line}
          organizationId={organizationId}
          canvasId={phase.appId ?? ""}
          spendPhaseId={phase.id}
          onNotes={reportNotes}
        />
      ))}
      {steps.length > 0 ? <AgentStepMarkers stage={{ ...shown, agentSteps: steps }} expandSteps={expandSteps} /> : null}
      {liveRunner ? (
        <LiveActivityScroller stageId={stage.id} tall={expandSteps}>
          <AgentLiveActivity
            active
            organizationId={organizationId}
            canvasId={phase.appId ?? ""}
            executionId={liveRunner.executionId ?? undefined}
            startingLabel="Starting agent…"
            testId={`redesign-live-activity-${stage.id}`}
          />
        </LiveActivityScroller>
      ) : steps.length === 0 ? (
        note
      ) : null}
    </>
  );
}

/** Pins the streaming activity to its newest line inside a capped scroller. */
function LiveActivityScroller({ stageId, tall, children }: { stageId: string; tall: boolean; children: ReactNode }) {
  const follow = useFollowLogScroll<HTMLDivElement>(stageId, null);
  return (
    <div className="relative min-w-0">
      <div
        ref={follow.scrollRef}
        onScroll={follow.onScroll}
        className={tall ? "max-h-[60vh] overflow-y-auto pl-4" : "max-h-80 overflow-y-auto pl-4"}
        data-testid={`redesign-live-scroll-${stageId}`}
      >
        {children}
      </div>
      {follow.showJumpToLatest ? (
        <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId={`redesign-live-older-${stageId}`} />
      ) : null}
    </div>
  );
}

function RunnerNotes({
  line,
  organizationId,
  canvasId,
  spendPhaseId,
  onNotes,
}: {
  line: SplitRunStreamLine;
  organizationId: string;
  canvasId: string;
  spendPhaseId: string;
  onNotes: (lineId: string, notes: SplitRunStreamLine[]) => void;
}) {
  const { notes, spend } = useRunnerLiveNotes(line, organizationId, canvasId);
  useReportLiveHeaderSpend(`${spendPhaseId}:${line.id}`, spend.tokens, spend.cents);
  useEffect(() => {
    onNotes(line.id, notes);
  }, [line.id, notes, onNotes]);
  return null;
}

function useRunnerLiveNotes(
  line: SplitRunStreamLine,
  organizationId: string,
  canvasId: string,
): { notes: SplitRunStreamLine[]; spend: ReturnType<typeof headerSpendFromUsageSeries> } {
  const canStream = Boolean(canvasId && line.executionId && isRunnerComponent(line.component));
  const { sections, orphanLines, error, isStreaming, usageSeries } = useLiveLogStream(
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
  return { notes, spend: headerSpendFromUsageSeries(usageSeries ?? []) };
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

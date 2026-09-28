import { useLiveLogStream } from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { useCallback, useEffect, useMemo, useState } from "react";

import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "../splitRunMocks";
import { isRunnerComponent, notesForLiveStream } from "../streamNotesFromLiveLog";
import { useSplitRunLiveCanvas } from "../useSplitRunLiveCanvas";
import { AgentStepMarkers } from "./AgentStepList";
import { agentStepsFromNotes, settleStoppedSteps, type AutomationStage } from "./automationsViewModel";

export function LiveAgentSteps({
  stage,
  phase,
  organizationId,
}: {
  stage: AutomationStage;
  phase?: SplitRunPhase;
  organizationId?: string;
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

  if (!organizationId || !phase || runners.length === 0) {
    const settled = { ...stage, agentSteps: settleStoppedSteps(stage.agentSteps, stoppedStatus) };
    return settled.agentSteps.length > 0 ? <AgentStepMarkers stage={settled} /> : null;
  }

  return (
    <>
      {runners.map((line) => (
        <RunnerNotes
          key={line.id}
          line={line}
          organizationId={organizationId}
          canvasId={phase.appId ?? ""}
          onNotes={reportNotes}
        />
      ))}
      {shown.agentSteps.length > 0 ? <AgentStepMarkers stage={shown} /> : null}
    </>
  );
}

function RunnerNotes({
  line,
  organizationId,
  canvasId,
  onNotes,
}: {
  line: SplitRunStreamLine;
  organizationId: string;
  canvasId: string;
  onNotes: (lineId: string, notes: SplitRunStreamLine[]) => void;
}) {
  const notes = useRunnerLiveNotes(line, organizationId, canvasId);
  useEffect(() => {
    onNotes(line.id, notes);
  }, [line.id, notes, onNotes]);
  return null;
}

function useRunnerLiveNotes(line: SplitRunStreamLine, organizationId: string, canvasId: string): SplitRunStreamLine[] {
  const canStream = Boolean(canvasId && line.executionId && isRunnerComponent(line.component));
  const { sections, orphanLines, error, isStreaming } = useLiveLogStream(
    canStream ? (line.executionId ?? "") : "",
    line.status === "running",
    liveLogFinishState(line.status),
    null,
    { organizationId, canvasId },
  );
  return useMemo(() => {
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

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { RunCanvasData } from "./useRunCanvasData";

export function useRunParticipantFitRequest({
  isRunInspectionMode,
  selectedRunId,
  hasSelectedRun,
  runCanvasLoading,
  runCanvasData,
}: {
  isRunInspectionMode: boolean;
  selectedRunId: string | null;
  hasSelectedRun: boolean;
  runCanvasLoading: boolean;
  runCanvasData: RunCanvasData | null;
}) {
  const nextRequestRef = useRef(0);
  const [fitIntent, setFitIntent] = useState<{ runId: string; requestId: number } | null>(null);
  const [fitRequest, setFitRequest] = useState<number | null>(null);
  const participantNodeIds =
    isRunInspectionMode && hasSelectedRun && runCanvasData?.participantNodeIds.length
      ? runCanvasData.participantNodeIds
      : undefined;
  const participantNodeIdsKey = participantNodeIds?.join("|") ?? "";

  const requestParticipantFit = useCallback((runId: string) => {
    nextRequestRef.current += 1;
    setFitIntent({ runId, requestId: nextRequestRef.current });
  }, []);

  const clearParticipantFit = useCallback(() => {
    setFitIntent(null);
    setFitRequest(null);
  }, []);

  useEffect(() => {
    if (!fitIntent || !isRunInspectionMode) return;
    if (fitIntent.runId !== selectedRunId) return;
    if (runCanvasLoading || !participantNodeIds?.length) return;

    setFitRequest(fitIntent.requestId);
    setFitIntent(null);
  }, [fitIntent, isRunInspectionMode, participantNodeIds, participantNodeIdsKey, runCanvasLoading, selectedRunId]);

  return useMemo(
    () => ({
      participantNodeIds,
      fitRequest,
      requestParticipantFit,
      clearParticipantFit,
    }),
    [clearParticipantFit, fitRequest, participantNodeIds, requestParticipantFit],
  );
}

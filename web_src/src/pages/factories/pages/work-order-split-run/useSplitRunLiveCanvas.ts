import { useCanvas, useDescribeRun } from "@/hooks/useCanvasData";
import { useCanvasRuntimeWebsocket } from "@/hooks/useCanvasWebsocket";
import { useMemo } from "react";

import { agentRunnerQueue, type AgentRunnerQueue } from "./agentRunnerQueue";
import { liveCanvasKeyForPhase, splitRunCanvasFromLive, streamFromLiveRun } from "./splitRunLiveCanvas";
import type { SplitRunCanvasModel } from "./splitRunCanvases";
import type { SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";

type SplitRunLiveCanvasResult = {
  enabled: boolean;
  isError: boolean;
  isLoading: boolean;
  canvas: SplitRunCanvasModel | undefined;
  stream: SplitRunStreamLine[];
  rootEventId: string | undefined;
  agentQueue?: AgentRunnerQueue;
};

function splitRunLiveQueryState(
  enabled: boolean,
  runId: string | null,
  canvasQuery: { isError: boolean; isLoading: boolean },
  runQuery: { isError: boolean; isLoading: boolean },
) {
  const runEnabled = Boolean(runId);
  return {
    isError: canvasQuery.isError || (runEnabled && runQuery.isError),
    isLoading: enabled && (canvasQuery.isLoading || (runEnabled && runQuery.isLoading)),
  };
}

export function useSplitRunLiveCanvas(
  organizationId: string | undefined,
  phase: SplitRunPhase | undefined,
): SplitRunLiveCanvasResult {
  const appId = phase?.appId ?? "";
  const runId = phase?.runId ?? null;
  const enabled = Boolean(organizationId && appId);
  useCanvasRuntimeWebsocket(appId, organizationId ?? "", enabled);
  const canvasQuery = useCanvas(organizationId ?? "", appId, { enabled });
  const runQuery = useDescribeRun(appId, runId, enabled && Boolean(runId));
  const { isError, isLoading } = splitRunLiveQueryState(enabled, runId, canvasQuery, runQuery);

  const canvas = useMemo(() => {
    if (!canvasQuery.data || !phase) {
      return undefined;
    }
    return splitRunCanvasFromLive({
      canvas: canvasQuery.data,
      run: runQuery.data?.run,
      fallbackTitle: phase.componentName,
      key: liveCanvasKeyForPhase(phase),
    });
  }, [canvasQuery.data, phase, runQuery.data?.run]);
  const stream = useMemo(
    () => streamFromLiveRun(canvasQuery.data, runQuery.data?.run),
    [canvasQuery.data, runQuery.data?.run],
  );
  const agentQueue = useMemo(() => {
    if (!enabled || !runId) {
      return undefined;
    }
    if (!canvasQuery.data || !runQuery.data?.run) {
      return undefined;
    }
    return agentRunnerQueue(canvasQuery.data, runQuery.data.run);
  }, [canvasQuery.data, enabled, runId, runQuery.data?.run]);

  return {
    enabled,
    isError,
    isLoading,
    canvas,
    stream,
    rootEventId: runQuery.data?.run?.rootEvent?.id,
    agentQueue,
  };
}

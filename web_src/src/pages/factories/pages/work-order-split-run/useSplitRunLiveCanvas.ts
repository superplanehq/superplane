import { useCanvas, useDescribeRun, useEventExecutions } from "@/hooks/useCanvasData";
import { useCanvasRuntimeWebsocket } from "@/hooks/useCanvasWebsocket";
import { useMemo } from "react";

import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";

import { agentRunnerQueue, type AgentRunnerQueue } from "./agentRunnerQueue";
import { liveCanvasKeyForPhase, splitRunCanvasFromLive, streamFromLiveRun } from "./splitRunLiveCanvas";
import type { SplitRunCanvasModel } from "./splitRunCanvases";
import type { SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";
import { isRunnerComponent } from "./streamNotesFromLiveLog";

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

function failedRunnerRootEventId(
  phase: SplitRunPhase | undefined,
  canvas: CanvasesCanvas | undefined,
  run: CanvasesCanvasRun | undefined,
): string | null {
  if (phase?.status !== "failed") {
    return null;
  }
  const nodes = canvas?.spec?.nodes ?? [];
  const runnerIds = new Set(nodes.filter((node) => isRunnerComponent(node.component)).map((node) => node.id));
  const hasFinishedRunner = run?.executions?.some(
    (execution) => execution.state === "STATE_FINISHED" && runnerIds.has(execution.nodeId),
  );
  if (!hasFinishedRunner) {
    return null;
  }
  return run?.rootEvent?.id ?? null;
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
  const run = runQuery.data?.run;
  const failedRootEventId = enabled ? failedRunnerRootEventId(phase, canvasQuery.data, run) : null;
  const executionsQuery = useEventExecutions(appId, failedRootEventId);
  const { isError, isLoading } = splitRunLiveQueryState(enabled, runId, canvasQuery, runQuery);

  const canvas = useMemo(() => {
    if (!canvasQuery.data || !phase) {
      return undefined;
    }
    return splitRunCanvasFromLive({
      canvas: canvasQuery.data,
      run,
      fallbackTitle: phase.componentName,
      key: liveCanvasKeyForPhase(phase),
    });
  }, [canvasQuery.data, phase, run]);
  const stream = useMemo(
    () => streamFromLiveRun(canvasQuery.data, run, failedRootEventId ? executionsQuery.data?.executions : []),
    [canvasQuery.data, executionsQuery.data?.executions, failedRootEventId, run],
  );
  const agentQueue = useMemo(() => {
    if (!enabled || !runId) {
      return undefined;
    }
    if (!canvasQuery.data || !run) {
      return undefined;
    }
    return agentRunnerQueue(canvasQuery.data, run);
  }, [canvasQuery.data, enabled, runId, run]);

  return {
    enabled,
    isError,
    isLoading: isLoading || executionsQuery.isLoading,
    canvas,
    stream,
    rootEventId: run?.rootEvent?.id,
    agentQueue,
  };
}

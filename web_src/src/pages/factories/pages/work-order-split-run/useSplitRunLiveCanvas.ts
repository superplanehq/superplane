import { useCanvas, useDescribeRun, useEventExecutions } from "@/hooks/useCanvasData";
import { useCanvasRuntimeWebsocket } from "@/hooks/useCanvasWebsocket";
import { useMemo } from "react";

import type { CanvasesCanvas, CanvasesCanvasRun, CanvasesCanvasNodeExecution } from "@/api-client";

import { agentRunnerQueue, type AgentRunnerQueue } from "./agentRunnerQueue";
import { liveCanvasKeyForPhase, splitRunCanvasFromLive, streamFromLiveRun } from "./splitRunLiveCanvas";
import type { SplitRunCanvasModel } from "./splitRunCanvases";
import type { SplitRunPhase, SplitRunStreamLine } from "./splitRunMocks";
import { isRunnerComponent } from "./streamNotesFromLiveLog";
import { toRunActivities } from "../automationRunHistory/runHistoryData";
import type { RunActivity } from "@/ui/AutomationRunHistory/types";

type SplitRunLiveCanvasResult = {
  enabled: boolean;
  isError: boolean;
  isLoading: boolean;
  canvas: SplitRunCanvasModel | undefined;
  stream: SplitRunStreamLine[];
  rootEventId: string | undefined;
  agentQueue?: AgentRunnerQueue;
  runnerActivities?: RunActivity[];
  fleetDetailsLoading?: boolean;
  fleetDetailsError?: boolean;
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
  const { appId, runId } = liveCanvasIdentifiers(phase);
  const enabled = Boolean(organizationId && appId);
  useCanvasRuntimeWebsocket(appId, organizationId ?? "", enabled);
  const canvasQuery = useCanvas(organizationId ?? "", appId, { enabled });
  const runQuery = useDescribeRun(appId, runId, enabled && Boolean(runId));
  const run = runQuery.data?.run;
  const failedRootEventId = enabled ? failedRunnerRootEventId(phase, canvasQuery.data, run) : null;
  const rootEventId = runnerExecutionRootEventId(enabled, runId, run);
  const executionsQuery = useEventExecutions(appId, rootEventId);
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
    runnerActivities: recordedRunnerActivities(run, executionsQuery.data?.executions, canvasQuery.data),
    fleetDetailsLoading: executionsQuery.isLoading,
    fleetDetailsError: executionsQuery.isError,
  };
}

function liveCanvasIdentifiers(phase: SplitRunPhase | undefined) {
  return { appId: phase?.appId ?? "", runId: phase?.runId ?? null };
}

function runnerExecutionRootEventId(enabled: boolean, runId: string | null, run: CanvasesCanvasRun | undefined) {
  if (!enabled || !runId) return null;
  return run?.rootEvent?.id ?? null;
}

function recordedRunnerActivities(
  run: CanvasesCanvasRun | undefined,
  executions: CanvasesCanvasNodeExecution[] | undefined,
  canvas: CanvasesCanvas | undefined,
) {
  if (!run) return [];
  return toRunActivities(run, executions ?? [], canvas?.spec?.nodes ?? []).filter(
    (activity) => activity.runnerFleet !== undefined,
  );
}

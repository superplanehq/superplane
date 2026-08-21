import { useCanvas, useDescribeRun } from "@/hooks/useCanvasData";
import { useMemo } from "react";

import { liveCanvasKeyForPhase, splitRunCanvasFromLive, streamFromLiveRun } from "./splitRunLiveCanvas";
import type { SplitRunPhase } from "./splitRunMocks";

export function useSplitRunLiveCanvas(organizationId: string | undefined, phase: SplitRunPhase | undefined) {
  const appId = phase?.appId ?? "";
  const runId = phase?.runId ?? null;
  const enabled = Boolean(organizationId && appId);
  const canvasQuery = useCanvas(organizationId ?? "", appId, { enabled });
  const runQuery = useDescribeRun(appId, runId, enabled && Boolean(runId));

  const canvas = useMemo(
    () =>
      canvasQuery.data && phase
        ? splitRunCanvasFromLive({
            canvas: canvasQuery.data,
            run: runQuery.data?.run,
            fallbackTitle: phase.componentName,
            key: liveCanvasKeyForPhase(phase),
          })
        : undefined,
    [canvasQuery.data, phase, runQuery.data?.run],
  );
  const stream = useMemo(
    () => streamFromLiveRun(canvasQuery.data, runQuery.data?.run),
    [canvasQuery.data, runQuery.data?.run],
  );

  return {
    enabled,
    isError: canvasQuery.isError || (Boolean(runId) && runQuery.isError),
    isLoading: enabled && (canvasQuery.isLoading || (Boolean(runId) && runQuery.isLoading)),
    canvas,
    stream,
  };
}

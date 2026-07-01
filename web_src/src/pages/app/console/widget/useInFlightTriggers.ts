import { useMemo } from "react";

import type { CanvasesCanvasRun } from "@/api-client";
import { useInfiniteCanvasRuns } from "@/hooks/useCanvasData";

/**
 * Returns the set of trigger node ids that currently have a `STATE_STARTED`
 * canvas run as the latest. Dashboard widgets use this signal to disable
 * row-action buttons whose trigger already has a pipeline executing, so the
 * user cannot enqueue duplicate runs while the canvas is still working.
 *
 * Implementation notes:
 * - We filter the runs query to `STATE_STARTED` to keep the payload small
 *   and the cache focused on the "still in flight" window.
 * - Correlation key is `run.rootEvent.nodeId === triggerNodeId`. The trigger
 *   API does not return a run id, so we rely on this back-reference (kept
 *   fresh by `run_started` / `run_finished` websocket invalidations via
 *   {@link useCanvasWebsocket}).
 * - The query is only enabled when there is at least one trigger node id to
 *   watch and a canvas id is available, so tables with no row actions skip
 *   the network call entirely.
 */
export function useInFlightTriggers(
  canvasId: string | undefined,
  triggerNodeIds: string[],
): { inFlight: Set<string>; isLoading: boolean } {
  const enabled = Boolean(canvasId) && triggerNodeIds.length > 0;
  const query = useInfiniteCanvasRuns(canvasId ?? "", { states: ["STATE_STARTED"] }, enabled);

  const inFlight = useMemo(() => {
    const out = new Set<string>();
    if (!query.data) return out;
    const watched = new Set(triggerNodeIds);
    for (const page of query.data.pages ?? []) {
      const runs: CanvasesCanvasRun[] = page?.runs ?? [];
      for (const run of runs) {
        if (run.state !== "STATE_STARTED") continue;
        const nodeId = run.rootEvent?.nodeId;
        if (nodeId && watched.has(nodeId)) out.add(nodeId);
      }
    }
    return out;
  }, [query.data, triggerNodeIds]);

  return { inFlight, isLoading: enabled && query.isLoading };
}

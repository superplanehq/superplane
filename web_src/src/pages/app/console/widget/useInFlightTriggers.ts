import { useMemo, useRef } from "react";

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

  // Re-using the previous Set when its content is unchanged keeps the
  // reference stable across renders. Downstream effects (e.g. the submission
  // lock in WidgetTableActionLock) depend on `inFlight` as a `useEffect` dep,
  // and recomputing a fresh Set on every render makes those effects fire on
  // every render, multiplying state-update work for tables with row actions.
  const inFlightRef = useRef<Set<string>>(new Set());

  const inFlight = useMemo(() => {
    const next = new Set<string>();
    if (query.data) {
      const watched = new Set(triggerNodeIds);
      for (const page of query.data.pages ?? []) {
        const runs: CanvasesCanvasRun[] = page?.runs ?? [];
        for (const run of runs) {
          if (run.state !== "STATE_STARTED") continue;
          const nodeId = run.rootEvent?.nodeId;
          if (nodeId && watched.has(nodeId)) next.add(nodeId);
        }
      }
    }
    if (areSetsEqual(inFlightRef.current, next)) {
      return inFlightRef.current;
    }
    inFlightRef.current = next;
    return next;
  }, [query.data, triggerNodeIds]);

  return { inFlight, isLoading: enabled && query.isLoading };
}

function areSetsEqual<T>(a: Set<T>, b: Set<T>): boolean {
  if (a === b) return true;
  if (a.size !== b.size) return false;
  for (const value of a) {
    if (!b.has(value)) return false;
  }
  return true;
}

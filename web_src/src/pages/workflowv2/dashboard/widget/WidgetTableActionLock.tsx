import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import { useDashboardContext } from "../DashboardContext";
import { useInFlightTriggers } from "./useInFlightTriggers";
import { WidgetTableActionLockReactContext, type WidgetTableActionLock } from "./WidgetTableActionLockContext";

const SUBMISSION_GRACE_MS = 1500;

/**
 * Wrap a `WidgetTable` body in this provider to share submission +
 * run-in-flight gating across every `RowActionButton` it renders.
 * `triggerNodeIds` should be the unique, resolved trigger node ids
 * referenced by the table's row actions — passing a non-empty list activates
 * the runs query.
 */
export function WidgetTableActionLockProvider({
  triggerNodeIds,
  children,
}: {
  triggerNodeIds: string[];
  children: ReactNode;
}) {
  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId;
  // `triggerNodeIds` is computed in the parent — caller is expected to memoize
  // it, but we defensively de-dupe + sort here so identity changes don't churn
  // the query / memo cache.
  const stableIds = useMemo(() => Array.from(new Set(triggerNodeIds)).sort(), [triggerNodeIds]);

  const { inFlight } = useInFlightTriggers(canvasId, stableIds);
  const lock = useSubmissionLock(inFlight);

  return (
    <WidgetTableActionLockReactContext.Provider value={lock}>{children}</WidgetTableActionLockReactContext.Provider>
  );
}

interface PendingEntry {
  rowKey: string;
  triggerNodeId: string | undefined;
}

/**
 * Tracks per-row submission state and the `triggerNodeId → rowKey` mapping
 * for runs initiated from this table. Each click adds the row key to
 * `pendingRowKeys` and (when a trigger node id is known) records the
 * mapping; the mapping persists while the trigger appears in
 * `runInFlightIds` so the originating row stays disabled across the gap
 * between the HTTP response and the websocket-driven runs refresh.
 *
 * Rows other than the originating one stay clickable even when the same
 * trigger has a run in flight — that's the "only the affected row" locking
 * model the console intentionally adopts.
 */
function useSubmissionLock(inFlightIds: Set<string>): WidgetTableActionLock {
  const [pendingRowKeys, setPendingRowKeys] = useState<Set<string>>(() => new Set());
  const [inFlightRowByTrigger, setInFlightRowByTrigger] = useState<Map<string, string>>(() => new Map());
  const timers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const pendingByKey = useRef<Map<string, PendingEntry>>(new Map());

  // When a trigger leaves the in-flight set (run completed or canceled), drop
  // its mapping so that next time the same trigger fires from any row the
  // gate logic starts fresh.
  useEffect(() => {
    setInFlightRowByTrigger((prev) => {
      let mutated = false;
      const next = new Map(prev);
      for (const trigger of prev.keys()) {
        if (!inFlightIds.has(trigger)) {
          next.delete(trigger);
          mutated = true;
        }
      }
      return mutated ? next : prev;
    });
  }, [inFlightIds]);

  // Once a recorded trigger actually shows up in `runInFlightIds`, clear its
  // pending row immediately — there is no need to keep the row in
  // `pendingRowKeys` because the mapping itself keeps the row disabled.
  useEffect(() => {
    if (pendingRowKeys.size === 0 || inFlightIds.size === 0) return;
    const toClear: string[] = [];
    for (const rowKey of pendingRowKeys) {
      const entry = pendingByKey.current.get(rowKey);
      if (entry?.triggerNodeId && inFlightIds.has(entry.triggerNodeId)) toClear.push(rowKey);
    }
    if (toClear.length === 0) return;
    setPendingRowKeys((prev) => {
      const next = new Set(prev);
      for (const key of toClear) {
        next.delete(key);
        const timer = timers.current.get(key);
        if (timer) {
          clearTimeout(timer);
          timers.current.delete(key);
        }
        pendingByKey.current.delete(key);
      }
      return next;
    });
  }, [inFlightIds, pendingRowKeys]);

  useEffect(() => {
    const timersMap = timers.current;
    return () => {
      for (const t of timersMap.values()) clearTimeout(t);
      timersMap.clear();
    };
  }, []);

  const beginSubmission = useCallback((triggerNodeId: string | undefined, rowKey: string) => {
    const existing = timers.current.get(rowKey);
    if (existing) {
      clearTimeout(existing);
      timers.current.delete(rowKey);
    }
    pendingByKey.current.set(rowKey, { rowKey, triggerNodeId });
    setPendingRowKeys((prev) => {
      if (prev.has(rowKey)) return prev;
      const next = new Set(prev);
      next.add(rowKey);
      return next;
    });
    if (triggerNodeId) {
      setInFlightRowByTrigger((prev) => {
        if (prev.get(triggerNodeId) === rowKey) return prev;
        const next = new Map(prev);
        next.set(triggerNodeId, rowKey);
        return next;
      });
    }
  }, []);

  const endSubmission = useCallback(
    (triggerNodeId: string | undefined, rowKey: string, succeeded: boolean) => {
      const removePending = () => {
        const timer = timers.current.get(rowKey);
        if (timer) {
          clearTimeout(timer);
          timers.current.delete(rowKey);
        }
        pendingByKey.current.delete(rowKey);
        setPendingRowKeys((prev) => {
          if (!prev.has(rowKey)) return prev;
          const next = new Set(prev);
          next.delete(rowKey);
          return next;
        });
      };

      if (!succeeded) {
        // Failed submission — the run never started, so drop both the
        // pending state and the optimistic mapping immediately.
        removePending();
        if (triggerNodeId) {
          setInFlightRowByTrigger((prev) => {
            if (prev.get(triggerNodeId) !== rowKey) return prev;
            const next = new Map(prev);
            next.delete(triggerNodeId);
            return next;
          });
        }
        return;
      }

      // Successful submission. If the runs query already sees this trigger
      // in flight, the mapping keeps the row disabled, so we can drop
      // pending right away. Otherwise wait one grace window for the
      // websocket-driven cache to catch up.
      if (triggerNodeId && inFlightIds.has(triggerNodeId)) {
        removePending();
        return;
      }
      const timer = setTimeout(() => {
        timers.current.delete(rowKey);
        pendingByKey.current.delete(rowKey);
        setPendingRowKeys((prev) => {
          if (!prev.has(rowKey)) return prev;
          const next = new Set(prev);
          next.delete(rowKey);
          return next;
        });
        // If the trigger never showed up in the in-flight set within the
        // grace window, also drop the optimistic mapping so the originating
        // row stops looking locked forever.
        if (triggerNodeId && !inFlightIds.has(triggerNodeId)) {
          setInFlightRowByTrigger((prev) => {
            if (prev.get(triggerNodeId) !== rowKey) return prev;
            const next = new Map(prev);
            next.delete(triggerNodeId);
            return next;
          });
        }
      }, SUBMISSION_GRACE_MS);
      timers.current.set(rowKey, timer);
    },
    [inFlightIds],
  );

  return useMemo<WidgetTableActionLock>(
    () => ({
      runInFlightIds: inFlightIds,
      pendingRowKeys,
      inFlightRowByTrigger,
      beginSubmission,
      endSubmission,
    }),
    [inFlightIds, pendingRowKeys, inFlightRowByTrigger, beginSubmission, endSubmission],
  );
}

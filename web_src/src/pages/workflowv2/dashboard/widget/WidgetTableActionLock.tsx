import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import { useDashboardContext } from "../DashboardContext";
import { useInFlightTriggers } from "./useInFlightTriggers";
import { WidgetTableActionLockReactContext, type WidgetTableActionLock } from "./WidgetTableActionLockContext";

const SUBMISSION_GRACE_MS = 1500;

/**
 * Wrap a `WidgetTable` body in this provider to share submission + run-in-flight
 * gating across every `RowActionButton` it renders. `triggerNodeIds` should be
 * the unique, resolved trigger node ids referenced by the table's row actions —
 * passing a non-empty list activates the runs query.
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

/**
 * Tracks the optimistic submission state: which trigger node ids the user
 * just clicked, and a grace timer that keeps the lock on for a short window
 * after `endSubmission` so the buttons stay disabled across the gap between
 * the HTTP response and the first websocket-driven runs query refresh.
 */
function useSubmissionLock(inFlightIds: Set<string>): WidgetTableActionLock {
  const [pendingIds, setPendingIds] = useState<Set<string>>(() => new Set());
  const timers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  useEffect(() => {
    if (pendingIds.size === 0 || inFlightIds.size === 0) return;
    let mutated = false;
    const next = new Set(pendingIds);
    for (const id of pendingIds) {
      if (inFlightIds.has(id)) {
        next.delete(id);
        const t = timers.current.get(id);
        if (t) {
          clearTimeout(t);
          timers.current.delete(id);
        }
        mutated = true;
      }
    }
    if (mutated) setPendingIds(next);
  }, [inFlightIds, pendingIds]);

  useEffect(() => {
    const timersMap = timers.current;
    return () => {
      for (const t of timersMap.values()) clearTimeout(t);
      timersMap.clear();
    };
  }, []);

  return useMemo<WidgetTableActionLock>(() => {
    const beginSubmission = (triggerNodeId: string | undefined) => {
      const key = triggerNodeId ?? "__unresolved__";
      const existing = timers.current.get(key);
      if (existing) {
        clearTimeout(existing);
        timers.current.delete(key);
      }
      setPendingIds((prev) => {
        if (prev.has(key)) return prev;
        const next = new Set(prev);
        next.add(key);
        return next;
      });
    };

    const endSubmission = (triggerNodeId: string | undefined) => {
      const key = triggerNodeId ?? "__unresolved__";
      if (triggerNodeId && inFlightIds.has(triggerNodeId)) {
        setPendingIds((prev) => {
          if (!prev.has(key)) return prev;
          const next = new Set(prev);
          next.delete(key);
          return next;
        });
        return;
      }
      const existing = timers.current.get(key);
      if (existing) clearTimeout(existing);
      const timer = setTimeout(() => {
        timers.current.delete(key);
        setPendingIds((prev) => {
          if (!prev.has(key)) return prev;
          const next = new Set(prev);
          next.delete(key);
          return next;
        });
      }, SUBMISSION_GRACE_MS);
      timers.current.set(key, timer);
    };

    return {
      runInFlightIds: inFlightIds,
      submitting: pendingIds.size > 0,
      beginSubmission,
      endSubmission,
    };
  }, [inFlightIds, pendingIds]);
}

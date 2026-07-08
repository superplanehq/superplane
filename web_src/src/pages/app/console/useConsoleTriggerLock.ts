import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type MutableRefObject,
  type SetStateAction,
} from "react";

import { useConsoleContext } from "./ConsoleContext";
import { useInFlightTriggers } from "./widget/useInFlightTriggers";

/**
 * How long the originating source stays locked after the invoke HTTP call
 * resolves, so the caller isn't temporarily re-enabled between "HTTP done"
 * and the runs query refreshing with `STATE_STARTED` on the same trigger.
 */
const SUBMISSION_GRACE_MS = 1500;

/**
 * Cross-widget lock covering two signals:
 *
 * 1. `runInFlightIds` — canvas run ids the backend currently reports in
 *    `STATE_STARTED`, keyed by trigger node id. Refreshed by the websocket-
 *    driven runs query in {@link useInFlightTriggers}.
 * 2. `pendingLockKeys` — arbitrary opaque keys (a table row key, a panel
 *    entry key, …) with a submission in flight. Kept live between click
 *    and the HTTP response plus a short grace window.
 * 3. `inFlightLockByTrigger` — `triggerNodeId → lockKey` mapping recorded on
 *    submission. Table widgets use it to lock *only* the originating row
 *    when the same trigger is fired again; node panels key submissions by
 *    the trigger node id itself, so every entry targeting that trigger in
 *    the panel's shared lock instance disables together.
 */
export interface ConsoleTriggerLock {
  runInFlightIds: ReadonlySet<string>;
  pendingLockKeys: ReadonlySet<string>;
  inFlightLockByTrigger: ReadonlyMap<string, string>;
  beginSubmission: (triggerNodeId: string | undefined, lockKey: string) => void;
  endSubmission: (triggerNodeId: string | undefined, lockKey: string, succeeded: boolean) => void;
}

interface UseConsoleTriggerLockArgs {
  /** Trigger node ids the caller cares about (drives the runs query). */
  triggerNodeIds: string[];
  /** Override the canvas id — defaults to the current console context. */
  canvasId?: string;
}

/**
 * Shared submission + in-flight lock for anywhere the console fires a
 * trigger — currently table row actions and the merged node panel. Lock
 * state is scoped to the hook instance, so create **one instance per
 * widget** and share it across that widget's buttons: the table does this
 * through {@link WidgetTableActionLockProvider}, and the node panel hoists
 * one instance per panel body. Per-button instances would not see each
 * other's pending submissions.
 */
export function useConsoleTriggerLock({ triggerNodeIds, canvasId }: UseConsoleTriggerLockArgs): ConsoleTriggerLock {
  const ctx = useConsoleContext();
  const effectiveCanvasId = canvasId ?? ctx?.canvasId;
  // Defensively de-dupe + sort so identity churn in the caller doesn't
  // remount the runs query or invalidate memoized dependencies.
  const stableIds = useMemo(() => Array.from(new Set(triggerNodeIds)).sort(), [triggerNodeIds]);
  const { inFlight } = useInFlightTriggers(effectiveCanvasId, stableIds);
  return useSubmissionLock(inFlight);
}

interface PendingEntry {
  lockKey: string;
  triggerNodeId: string | undefined;
}

type PendingTimers = MutableRefObject<Map<string, ReturnType<typeof setTimeout>>>;
type PendingByKey = MutableRefObject<Map<string, PendingEntry>>;

function clearPendingTimer(lockKey: string, timers: PendingTimers) {
  const timer = timers.current.get(lockKey);
  if (timer) {
    clearTimeout(timer);
    timers.current.delete(lockKey);
  }
}

function dropPending(
  lockKey: string,
  timers: PendingTimers,
  pendingByKey: PendingByKey,
  setPendingLockKeys: Dispatch<SetStateAction<Set<string>>>,
) {
  clearPendingTimer(lockKey, timers);
  pendingByKey.current.delete(lockKey);
  setPendingLockKeys((prev) => {
    if (!prev.has(lockKey)) return prev;
    const next = new Set(prev);
    next.delete(lockKey);
    return next;
  });
}

function clearTriggerMapping(
  triggerNodeId: string,
  lockKey: string,
  setInFlightLockByTrigger: Dispatch<SetStateAction<Map<string, string>>>,
) {
  setInFlightLockByTrigger((prev) => {
    if (prev.get(triggerNodeId) !== lockKey) return prev;
    const next = new Map(prev);
    next.delete(triggerNodeId);
    return next;
  });
}

/**
 * Track per-source submission state and the `triggerNodeId → lockKey`
 * mapping for runs initiated from this widget. Each `beginSubmission` adds
 * the lock key to `pendingLockKeys` and (when a trigger node id is known)
 * records the mapping; the mapping persists while the trigger appears in
 * `runInFlightIds` so the originating source stays disabled across the gap
 * between the HTTP response and the websocket-driven runs refresh.
 */
function useSubmissionLock(inFlightIds: ReadonlySet<string>): ConsoleTriggerLock {
  const [pendingLockKeys, setPendingLockKeys] = useState<Set<string>>(() => new Set());
  const [inFlightLockByTrigger, setInFlightLockByTrigger] = useState<Map<string, string>>(() => new Map());
  const timers = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  const pendingByKey = useRef<Map<string, PendingEntry>>(new Map());
  // Grace timers outlive `endSubmission`; read the latest websocket-driven
  // in-flight set when they fire instead of the snapshot from submit time.
  const inFlightIdsRef = useRef(inFlightIds);
  inFlightIdsRef.current = inFlightIds;

  useEffect(() => {
    setInFlightLockByTrigger((prev) => {
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

  useEffect(() => {
    if (pendingLockKeys.size === 0 || inFlightIds.size === 0) return;
    const toClear: string[] = [];
    for (const lockKey of pendingLockKeys) {
      const entry = pendingByKey.current.get(lockKey);
      if (entry?.triggerNodeId && inFlightIds.has(entry.triggerNodeId)) toClear.push(lockKey);
    }
    if (toClear.length === 0) return;
    setPendingLockKeys((prev) => {
      const next = new Set(prev);
      for (const key of toClear) {
        next.delete(key);
        clearPendingTimer(key, timers);
        pendingByKey.current.delete(key);
      }
      return next;
    });
  }, [inFlightIds, pendingLockKeys]);

  useEffect(() => {
    const timersMap = timers.current;
    return () => {
      for (const t of timersMap.values()) clearTimeout(t);
      timersMap.clear();
    };
  }, []);

  const beginSubmission = useCallback((triggerNodeId: string | undefined, lockKey: string) => {
    clearPendingTimer(lockKey, timers);
    pendingByKey.current.set(lockKey, { lockKey, triggerNodeId });
    setPendingLockKeys((prev) => {
      if (prev.has(lockKey)) return prev;
      const next = new Set(prev);
      next.add(lockKey);
      return next;
    });
    if (triggerNodeId) {
      setInFlightLockByTrigger((prev) => {
        if (prev.get(triggerNodeId) === lockKey) return prev;
        const next = new Map(prev);
        next.set(triggerNodeId, lockKey);
        return next;
      });
    }
  }, []);

  const endSubmission = useCallback((triggerNodeId: string | undefined, lockKey: string, succeeded: boolean) => {
    if (!succeeded) {
      dropPending(lockKey, timers, pendingByKey, setPendingLockKeys);
      if (triggerNodeId) clearTriggerMapping(triggerNodeId, lockKey, setInFlightLockByTrigger);
      return;
    }

    if (triggerNodeId && inFlightIdsRef.current.has(triggerNodeId)) {
      dropPending(lockKey, timers, pendingByKey, setPendingLockKeys);
      return;
    }

    const timer = setTimeout(() => {
      dropPending(lockKey, timers, pendingByKey, setPendingLockKeys);
      if (triggerNodeId && !inFlightIdsRef.current.has(triggerNodeId)) {
        clearTriggerMapping(triggerNodeId, lockKey, setInFlightLockByTrigger);
      }
    }, SUBMISSION_GRACE_MS);
    timers.current.set(lockKey, timer);
  }, []);

  return useMemo<ConsoleTriggerLock>(
    () => ({
      runInFlightIds: inFlightIds,
      pendingLockKeys,
      inFlightLockByTrigger,
      beginSubmission,
      endSubmission,
    }),
    [inFlightIds, pendingLockKeys, inFlightLockByTrigger, beginSubmission, endSubmission],
  );
}

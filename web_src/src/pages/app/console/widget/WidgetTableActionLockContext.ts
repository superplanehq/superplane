import { createContext, useContext } from "react";

import type { ConsoleTriggerLock } from "../useConsoleTriggerLock";

/**
 * Per-table action lock state. Combines the same three signals as
 * {@link ConsoleTriggerLock} but re-exposed under the table's original
 * row-key terminology so the existing table code (and its specs) keep
 * their names:
 *
 * 1. `runInFlightIds` — trigger node ids whose latest canvas run is currently
 *    in `STATE_STARTED`. Driven by the runs query + canvas websocket.
 * 2. `pendingRowKeys` — set of row keys with a submission in flight (between
 *    click and HTTP response, plus a short grace window). Locking is
 *    scoped per row so siblings stay clickable while one row is submitting.
 * 3. `inFlightRowByTrigger` — `triggerNodeId → rowKey` mapping recorded on
 *    submission so the row that started the run is the only one locked
 *    while the trigger is in flight. Rows whose trigger appears in
 *    `runInFlightIds` without a recorded source rowKey are NOT locked, by
 *    design: the user explicitly asked for "only the affected row" locking.
 */
export interface WidgetTableActionLock {
  runInFlightIds: ReadonlySet<string>;
  pendingRowKeys: ReadonlySet<string>;
  inFlightRowByTrigger: ReadonlyMap<string, string>;
  beginSubmission: (triggerNodeId: string | undefined, rowKey: string) => void;
  endSubmission: (triggerNodeId: string | undefined, rowKey: string, succeeded: boolean) => void;
}

export const WidgetTableActionLockReactContext = createContext<WidgetTableActionLock | undefined>(undefined);

const FALLBACK_LOCK: WidgetTableActionLock = {
  runInFlightIds: new Set(),
  pendingRowKeys: new Set(),
  inFlightRowByTrigger: new Map(),
  beginSubmission: () => {},
  endSubmission: () => {},
};

/**
 * Read the per-table action lock. Returns a no-op lock when used outside a
 * `WidgetTableActionLockProvider` so consumers (e.g. the canvas node action
 * chips, which reuse `RowActionButton` mechanics) keep working unchanged.
 */
export function useWidgetTableActionLock(): WidgetTableActionLock {
  return useContext(WidgetTableActionLockReactContext) ?? FALLBACK_LOCK;
}

/**
 * Adapt a generic {@link ConsoleTriggerLock} to the row-scoped
 * {@link WidgetTableActionLock} shape. The two structures are identical up
 * to naming — this just re-exposes the fields under the table's original
 * `pendingRowKeys` / `inFlightRowByTrigger` labels so the existing table
 * code keeps its intent-revealing names.
 */
export function widgetTableLockFromConsoleLock(lock: ConsoleTriggerLock): WidgetTableActionLock {
  return {
    runInFlightIds: lock.runInFlightIds,
    pendingRowKeys: lock.pendingLockKeys,
    inFlightRowByTrigger: lock.inFlightLockByTrigger,
    beginSubmission: lock.beginSubmission,
    endSubmission: lock.endSubmission,
  };
}

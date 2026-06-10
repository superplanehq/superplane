import { createContext, useContext } from "react";

/**
 * Per-table action lock state. Combines two signals so authors cannot fire
 * a duplicate trigger while a pipeline is still running:
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
  runInFlightIds: Set<string>;
  pendingRowKeys: Set<string>;
  inFlightRowByTrigger: Map<string, string>;
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

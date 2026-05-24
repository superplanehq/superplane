import { createContext, useContext } from "react";

/**
 * Per-table action lock state. Combines two signals so authors cannot fire
 * a duplicate trigger while a pipeline is still running:
 *
 * 1. `runInFlightIds` — trigger node ids whose latest canvas run is currently
 *    in `STATE_STARTED`. Driven by the runs query + canvas websocket.
 * 2. `submitting` — true between the moment a row-action click calls
 *    `beginSubmission()` and the moment the runs query has confirmed the new
 *    run is in flight (or a short grace window has elapsed, whichever comes
 *    first). This covers the small gap between the `InvokeNodeTriggerHook`
 *    API returning and the websocket landing a `run_started` event, so the
 *    buttons do not briefly flicker back to enabled.
 */
export interface WidgetTableActionLock {
  runInFlightIds: Set<string>;
  submitting: boolean;
  beginSubmission: (triggerNodeId: string | undefined) => void;
  endSubmission: (triggerNodeId: string | undefined) => void;
}

export const WidgetTableActionLockReactContext = createContext<WidgetTableActionLock | undefined>(undefined);

const FALLBACK_LOCK: WidgetTableActionLock = {
  runInFlightIds: new Set(),
  submitting: false,
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

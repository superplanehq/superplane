import { useMemo, type ReactNode } from "react";

import { useConsoleTriggerLock } from "../useConsoleTriggerLock";
import { WidgetTableActionLockReactContext, widgetTableLockFromConsoleLock } from "./WidgetTableActionLockContext";

/**
 * Wrap a `WidgetTable` body in this provider to share submission +
 * run-in-flight gating across every `RowActionButton` it renders.
 * `triggerNodeIds` should be the unique, resolved trigger node ids
 * referenced by the table's row actions — passing a non-empty list activates
 * the runs query.
 *
 * The provider is a thin adapter over the shared
 * {@link useConsoleTriggerLock}, re-exposing the generic lock under the
 * table's original row-key terminology so existing consumers stay
 * unchanged. Every other trigger-firing widget consumes the same underlying
 * hook directly (see `useConsoleRunTrigger`).
 */
export function WidgetTableActionLockProvider({
  triggerNodeIds,
  children,
}: {
  triggerNodeIds: string[];
  children: ReactNode;
}) {
  const consoleLock = useConsoleTriggerLock({ triggerNodeIds });
  const value = useMemo(() => widgetTableLockFromConsoleLock(consoleLock), [consoleLock]);
  return (
    <WidgetTableActionLockReactContext.Provider value={value}>{children}</WidgetTableActionLockReactContext.Provider>
  );
}

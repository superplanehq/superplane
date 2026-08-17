import { useCallback, useMemo, useState, type ReactNode } from "react";

import type { WorkOrderReactionGroup } from "../../workOrderReactionTypes";
import { CURRENT_REACTOR_NAME, SEEDED_WORK_ORDER_REACTIONS } from "./reactionMocks";
import { toggleWorkOrderReaction } from "./reactionModel";
import { WorkOrderReactionsStoreContext } from "./reactionStoreContextValue";

/** Storybook-only in-memory reaction store, keyed by work order id. Not sent to the work-order API. */
export function WorkOrderReactionsProvider({ children }: { children: ReactNode }) {
  const [reactionsByWorkOrderId, setReactionsByWorkOrderId] = useState<Record<string, WorkOrderReactionGroup[]>>(
    () => ({ ...SEEDED_WORK_ORDER_REACTIONS }),
  );

  const toggleReaction = useCallback((workOrderId: string, emoji: string) => {
    setReactionsByWorkOrderId((current) => ({
      ...current,
      [workOrderId]: toggleWorkOrderReaction(current[workOrderId] ?? [], emoji, CURRENT_REACTOR_NAME),
    }));
  }, []);

  const value = useMemo(() => ({ reactionsByWorkOrderId, toggleReaction }), [reactionsByWorkOrderId, toggleReaction]);

  return <WorkOrderReactionsStoreContext.Provider value={value}>{children}</WorkOrderReactionsStoreContext.Provider>;
}

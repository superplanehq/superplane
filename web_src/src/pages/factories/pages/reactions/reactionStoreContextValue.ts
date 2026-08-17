import { createContext } from "react";

import type { WorkOrderReactionGroup } from "../../workOrderReactionTypes";

export interface WorkOrderReactionsStoreValue {
  reactionsByWorkOrderId: Record<string, WorkOrderReactionGroup[]>;
  toggleReaction: (workOrderId: string, emoji: string) => void;
}

export const WorkOrderReactionsStoreContext = createContext<WorkOrderReactionsStoreValue | null>(null);

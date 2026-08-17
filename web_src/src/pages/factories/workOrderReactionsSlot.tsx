import { createContext, useContext, type ComponentType } from "react";

/**
 * Storybook can mount a reaction bar under the work order title. The live
 * app leaves this empty until reactions are wired to a real API. Same
 * pattern as `WorkOrderOverviewMissionSlotContext`.
 */
export type WorkOrderReactionsSlot = ComponentType<{ workOrderId: string }>;

export const WorkOrderReactionsSlotContext = createContext<WorkOrderReactionsSlot | null>(null);

export function useWorkOrderReactionsSlot() {
  return useContext(WorkOrderReactionsSlotContext);
}

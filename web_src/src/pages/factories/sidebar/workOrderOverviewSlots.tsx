import { createContext, useContext, type ComponentType } from "react";

/** Storybook can mount a Mission row in Overview. The live app leaves this empty. */
export type WorkOrderOverviewMissionSlot = ComponentType<{ workOrderId: string }>;

export const WorkOrderOverviewMissionSlotContext = createContext<WorkOrderOverviewMissionSlot | null>(null);

export function useWorkOrderOverviewMissionSlot() {
  return useContext(WorkOrderOverviewMissionSlotContext);
}

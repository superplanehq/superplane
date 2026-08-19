import { createContext, useContext } from "react";

import type { WorkOrderCheckPresentation } from "./lib/workOrderChecks";

/**
 * Storybook can attach boolean checks (CI, security scan, …) to a work
 * order's Checks section, next to the real scored checks. The live app
 * provides no value for this context, so `useWorkOrderChecksPrototypeSlot`
 * returns `null` there and the section renders unchanged.
 */
export type WorkOrderChecksPrototypeSlot = (workOrderId: string) => WorkOrderCheckPresentation[];

export const WorkOrderChecksPrototypeSlotContext = createContext<WorkOrderChecksPrototypeSlot | null>(null);

export function useWorkOrderChecksPrototypeSlot() {
  return useContext(WorkOrderChecksPrototypeSlotContext);
}

import { createContext, useContext, type ComponentType } from "react";

/**
 * Storybook can mount a velocity scorecard row at the top of Overview. The
 * live app leaves this empty, so production Overview is unchanged.
 */
export type OverviewMetricsSlot = ComponentType<{ organizationId: string; factoryKey: string }>;

export const OverviewMetricsSlotContext = createContext<OverviewMetricsSlot | null>(null);

export function useOverviewMetricsSlot() {
  return useContext(OverviewMetricsSlotContext);
}

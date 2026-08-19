import { createContext, useContext } from "react";

import { formatDurationHours } from "./factoryVelocityFlow";
import type { VelocityDurationFormat } from "./velocityDurationFormat";

function defaultFormatTick(hours: number): string {
  return `${Number(hours).toFixed(0)}h`;
}

/** Matches today's live behavior: whole-hour rounding, a single "h" axis unit. */
const defaultVelocityDurationFormat: VelocityDurationFormat = {
  formatDuration: formatDurationHours,
  pickChartUnit: () => ({ unit: "h", formatTick: defaultFormatTick }),
};

/**
 * Storybook can swap in a sub-hour-aware duration formatter for the Velocity
 * work-order time scorecards and the Time running / Time in Waiting chart.
 * The live app leaves this empty, so consumers fall back to today's
 * whole-hour rounding.
 */
export const VelocityDurationFormatSlotContext = createContext<VelocityDurationFormat | null>(null);

export function useVelocityDurationFormat(): VelocityDurationFormat {
  return useContext(VelocityDurationFormatSlotContext) ?? defaultVelocityDurationFormat;
}

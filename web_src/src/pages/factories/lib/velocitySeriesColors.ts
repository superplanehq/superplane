/**
 * One color per Velocity series. Charts and the metric cells next to them share
 * these values, so a number and its line or bar always carry the same color.
 */
export const VELOCITY_SERIES_COLORS = {
  merged: "#10b981",
  waste: "#ef4444",
  cost: "#64748b",
  people: "#64748b",
  superplane: "#10b981",
  running: "#60a5fa",
  waiting: "#f59e0b",
} as const;

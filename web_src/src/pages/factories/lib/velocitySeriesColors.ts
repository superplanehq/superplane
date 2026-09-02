/**
 * One color per Velocity series. Charts and the metric cells next to them share
 * these values, so a number and its line or bar always carry the same color.
 */

export const VELOCITY_ORIGIN_COLORS = {
  people: "#64748b",
  superplane: "#10b981",
} as const;

export const VELOCITY_OUTCOME_COLORS = {
  merged: "#10b981",
  waste: "#ef4444",
} as const;

export const VELOCITY_TIME_COLORS = {
  running: "#60a5fa",
  waiting: "#f59e0b",
} as const;

export const VELOCITY_COST_COLOR = "#6366f1";

/**
 * Colors of the intake bands. Known sources keep one color across workspaces so
 * a reader who learns the chart once keeps reading it. Sources added later fall
 * back to the spare palette.
 */
const INTAKE_COLORS: Record<string, string> = {
  "github-issues": "#3b82f6",
  "sentry-exceptions": "#8b5cf6",
  "pagerduty-incidents": "#f97316",
  imported: "#06b6d4",
  manual: "#f59e0b",
  automation: "#14b8a6",
};

const SPARE_INTAKE_COLORS = ["#0ea5e9", "#a855f7", "#ec4899", "#84cc16"];

export function velocityIntakeColor(key: string, index: number): string {
  return INTAKE_COLORS[key] ?? SPARE_INTAKE_COLORS[index % SPARE_INTAKE_COLORS.length];
}

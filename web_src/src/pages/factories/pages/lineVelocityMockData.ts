export type VelocityPeriodDays = 7 | 30 | 90;

export type VelocityPeriodStats = {
  days: VelocityPeriodDays;
  label: string;
  runs: number;
  succeeded: number;
  failed: number;
  /** Average tokens per completed run in the period. */
  tokensPerRun: number;
  /** Estimated token spend per completed run (USD). */
  tokenSpendPerRun: number;
  /** Estimated VM spend per completed run (USD). */
  vmSpendPerRun: number;
};

/** Storybook / UI mock stats by selected window. */
export const VELOCITY_BY_PERIOD: Record<VelocityPeriodDays, VelocityPeriodStats> = {
  7: {
    days: 7,
    label: "Last 7 days",
    runs: 42,
    succeeded: 28,
    failed: 5,
    tokensPerRun: 18400,
    tokenSpendPerRun: 0.42,
    vmSpendPerRun: 1.18,
  },
  30: {
    days: 30,
    label: "Last 30 days",
    runs: 168,
    succeeded: 121,
    failed: 22,
    tokensPerRun: 17200,
    tokenSpendPerRun: 0.39,
    vmSpendPerRun: 1.05,
  },
  90: {
    days: 90,
    label: "Last 90 days",
    runs: 512,
    succeeded: 381,
    failed: 64,
    tokensPerRun: 16900,
    tokenSpendPerRun: 0.37,
    vmSpendPerRun: 0.98,
  },
};

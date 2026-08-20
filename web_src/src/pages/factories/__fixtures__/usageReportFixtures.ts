/** Storybook payload for factory Usage and org LLM spend reports. */
export interface StorybookUsageReport {
  totalTokens: string;
  totalCostCents: string;
  periodDays: number;
  byModel: Array<{ provider: string; model: string; totalTokens: string; costCents: string }>;
}

export const EMPTY_USAGE_REPORT: StorybookUsageReport = {
  totalTokens: "0",
  totalCostCents: "0",
  periodDays: 30,
  byModel: [],
};

/** Totals match spend on the populated Refunds Factory work orders. */
export const DEFAULT_FACTORY_USAGE: StorybookUsageReport = {
  totalTokens: "25600",
  totalCostCents: "876",
  periodDays: 30,
  byModel: [
    { provider: "anthropic", model: "claude-sonnet-4-6", totalTokens: "18400", costCents: "620" },
    { provider: "openai", model: "gpt-4o", totalTokens: "7200", costCents: "256" },
  ],
};

/** Storybook payload for factory Usage and org LLM spend reports. */
export interface StorybookUsageReport {
  totalTokens: string;
  totalCostCents: string;
  periodDays: number;
  byModel: Array<{ provider: string; model: string; totalTokens: string; costCents: string }>;
  remainingCreditCents?: string;
  grantTotalCents?: string;
  superplaneGrantCents?: string;
  purchasedCreditCents?: string;
  hostedBilledCents?: string;
  remainingCreditWarning?: boolean;
  billingEnabled?: boolean;
  hasBillingCustomer?: boolean;
  invoices?: Array<{
    id?: string;
    createdAt?: string;
    amountCents?: string;
    status?: string;
    productName?: string;
  }>;
  hostedSpendBudgetCents?: string | number | null;
  factoryHostedBilledCents?: string;
  factoryRemainingCreditCents?: string;
  factoryRemainingCreditWarning?: boolean;
}

export const EMPTY_USAGE_REPORT: StorybookUsageReport = {
  totalTokens: "0",
  totalCostCents: "0",
  periodDays: 30,
  byModel: [],
  remainingCreditCents: "5000",
  grantTotalCents: "5000",
  superplaneGrantCents: "5000",
  purchasedCreditCents: "0",
  hostedBilledCents: "0",
  remainingCreditWarning: false,
};

export const NO_GRANT_USAGE_REPORT: StorybookUsageReport = {
  totalTokens: "0",
  totalCostCents: "0",
  periodDays: 30,
  byModel: [],
  remainingCreditCents: "0",
  grantTotalCents: "0",
  superplaneGrantCents: "0",
  purchasedCreditCents: "0",
  hostedBilledCents: "0",
  remainingCreditWarning: false,
};

/** Totals match spend on the populated Refunds Factory tasks. */
export const DEFAULT_FACTORY_USAGE: StorybookUsageReport = {
  totalTokens: "25600",
  totalCostCents: "876",
  periodDays: 30,
  byModel: [
    { provider: "anthropic", model: "claude-sonnet-4-6", totalTokens: "18400", costCents: "620" },
    { provider: "openai", model: "gpt-4o", totalTokens: "7200", costCents: "256" },
  ],
  remainingCreditCents: "4124",
  grantTotalCents: "5000",
  superplaneGrantCents: "5000",
  purchasedCreditCents: "0",
  hostedBilledCents: "876",
  remainingCreditWarning: false,
};

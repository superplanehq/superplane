/** Storybook payload for factory Usage and org workspace usage reports. */
export interface StorybookUsageReport {
  totalTokens: string;
  totalCostCents: string;
  totalDurationSeconds?: string;
  periodDays: number;
  byModel: Array<{ provider: string; model: string; totalTokens: string; costCents: string }>;
  byMachineType?: Array<{ machineType: string; durationSeconds: string; costCents: string }>;
  remainingCreditCents?: string;
  grantTotalCents?: string;
  superplaneGrantCents?: string;
  purchasedCreditCents?: string;
  hostedBilledCents?: string;
  remainingCreditWarning?: boolean;
  billingEnabled?: boolean;
  hasBillingCustomer?: boolean;
  welcomeCreditExpiresAt?: string;
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
  totalDurationSeconds: "0",
  periodDays: 30,
  byModel: [],
  byMachineType: [],
  remainingCreditCents: "5000",
  grantTotalCents: "5000",
  superplaneGrantCents: "5000",
  purchasedCreditCents: "0",
  hostedBilledCents: "0",
  remainingCreditWarning: false,
  welcomeCreditExpiresAt: "2026-09-22T12:00:00.000Z",
};

export const NO_GRANT_USAGE_REPORT: StorybookUsageReport = {
  totalTokens: "0",
  totalCostCents: "0",
  totalDurationSeconds: "0",
  periodDays: 30,
  byModel: [],
  byMachineType: [],
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
  totalDurationSeconds: "3600",
  periodDays: 30,
  byModel: [
    { provider: "anthropic", model: "claude-sonnet-4-6", totalTokens: "18400", costCents: "620" },
    { provider: "openai", model: "gpt-4o", totalTokens: "7200", costCents: "256" },
  ],
  byMachineType: [
    { machineType: "e1-large-amd64", durationSeconds: "2400", costCents: "133" },
    { machineType: "e1-tiny-amd64", durationSeconds: "1200", costCents: "17" },
  ],
  remainingCreditCents: "4124",
  grantTotalCents: "5000",
  superplaneGrantCents: "5000",
  purchasedCreditCents: "0",
  hostedBilledCents: "876",
  remainingCreditWarning: false,
  welcomeCreditExpiresAt: "2026-09-22T12:00:00.000Z",
};

/** Welcome grant spent. Remaining hosted credit is empty. Polar recovery is available. */
export const SPENT_CREDIT_USAGE_REPORT: StorybookUsageReport = {
  ...DEFAULT_FACTORY_USAGE,
  remainingCreditCents: "0",
  grantTotalCents: "5000",
  superplaneGrantCents: "5000",
  purchasedCreditCents: "0",
  hostedBilledCents: "5000",
  remainingCreditWarning: true,
  billingEnabled: true,
  hasBillingCustomer: true,
};

/** Welcome credit remains, but the balance is at or below $20. */
export const LOW_TRIAL_USAGE_REPORT: StorybookUsageReport = {
  ...DEFAULT_FACTORY_USAGE,
  remainingCreditCents: "432",
  hostedBilledCents: "4568",
  remainingCreditWarning: true,
};

export const EXPIRED_WELCOME_USAGE_REPORT: StorybookUsageReport = {
  ...SPENT_CREDIT_USAGE_REPORT,
  welcomeCreditExpiresAt: "2026-08-15T12:00:00.000Z",
};

export const PURCHASED_CREDIT_USAGE_REPORT: StorybookUsageReport = {
  ...DEFAULT_FACTORY_USAGE,
  remainingCreditCents: "14124",
  grantTotalCents: "15000",
  purchasedCreditCents: "10000",
  welcomeCreditExpiresAt: "2026-09-22T12:00:00.000Z",
};

/** Purchased hosted credit remains, but the balance is at or below $20. */
export const LOW_CREDIT_USAGE_REPORT: StorybookUsageReport = {
  ...PURCHASED_CREDIT_USAGE_REPORT,
  remainingCreditCents: "1500",
  hostedBilledCents: "13500",
  remainingCreditWarning: true,
  billingEnabled: true,
  hasBillingCustomer: true,
};

/** Polar packs for empty-credit Storybook recovery screens. */
export const STORYBOOK_HOSTED_CREDIT_PRODUCTS = [
  { id: "prod-50", name: "Hosted credit 50", amountCents: "5000" },
  { id: "prod-100", name: "Hosted credit 100", amountCents: "10000" },
  { id: "prod-500", name: "Hosted credit 500", amountCents: "50000" },
];

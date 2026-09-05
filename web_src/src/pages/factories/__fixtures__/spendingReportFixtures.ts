import { PRIMARY_FACTORY_ID } from "./factoryPageIds";
import {
  SPENDING_CATALOGS,
  SPENDING_CREDIT,
} from "../pages/organizationSettings/spending-redesign/spendingRedesignMocks";

/** Storybook payload for GET /organizations/{id}/spending-report. */
export interface StorybookSpendingReport {
  kpiTotals: {
    costCents: string;
    totalTokens: string;
    durationSeconds: string;
    hostedCostCents: string;
    byokCostCents: string;
  };
  explorerTotals: {
    costCents: string;
    totalTokens: string;
    durationSeconds: string;
  };
  series: Array<{
    key: string;
    label: string;
    totalCents: string;
    values: Array<{ seriesId: string; costCents: string }>;
  }>;
  seriesKeys: Array<{ id: string; label: string }>;
  breakdown: Array<{
    id: string;
    label: string;
    totalTokens: string;
    durationSeconds: string;
    costCents: string;
    share: number;
  }>;
  credit: {
    remainingCreditCents: string;
    grantTotalCents: string;
    superplaneGrantCents: string;
    purchasedCreditCents: string;
    hostedBilledCents: string;
    remainingCreditWarning: boolean;
    billingEnabled: boolean;
    hasBillingCustomer: boolean;
  };
  catalogs: {
    workspaces: Array<{ id: string; label: string }>;
    users: Array<{ id: string; label: string }>;
    models: Array<{ id: string; label: string }>;
    machines: Array<{ id: string; label: string }>;
  };
}

export const DEFAULT_ORG_SPENDING_REPORT: StorybookSpendingReport = {
  kpiTotals: {
    costCents: "10876",
    totalTokens: "4820000",
    durationSeconds: "93600",
    hostedCostCents: "9200",
    byokCostCents: "1676",
  },
  explorerTotals: {
    costCents: "6200",
    totalTokens: "3200000",
    durationSeconds: "0",
  },
  series: [
    {
      key: "2026-08-04",
      label: "Aug 4",
      totalCents: "2100",
      values: [{ seriesId: PRIMARY_FACTORY_ID, costCents: "2100" }],
    },
    {
      key: "2026-08-05",
      label: "Aug 5",
      totalCents: "4100",
      values: [{ seriesId: PRIMARY_FACTORY_ID, costCents: "4100" }],
    },
  ],
  seriesKeys: [{ id: PRIMARY_FACTORY_ID, label: "Semaphore" }],
  breakdown: [
    {
      id: PRIMARY_FACTORY_ID,
      label: "Semaphore",
      totalTokens: "3200000",
      durationSeconds: "0",
      costCents: "6200",
      share: 1,
    },
  ],
  credit: {
    remainingCreditCents: String(SPENDING_CREDIT.remainingCreditCents),
    grantTotalCents: String(SPENDING_CREDIT.grantTotalCents),
    superplaneGrantCents: String(SPENDING_CREDIT.superplaneGrantCents),
    purchasedCreditCents: String(SPENDING_CREDIT.purchasedCreditCents),
    hostedBilledCents: String(SPENDING_CREDIT.hostedBilledCents),
    remainingCreditWarning: SPENDING_CREDIT.remainingCreditWarning,
    billingEnabled: SPENDING_CREDIT.billingEnabled,
    hasBillingCustomer: SPENDING_CREDIT.hasBillingCustomer,
  },
  catalogs: {
    workspaces: SPENDING_CATALOGS.workspaces,
    users: SPENDING_CATALOGS.users,
    models: SPENDING_CATALOGS.models,
    machines: SPENDING_CATALOGS.machines,
  },
};

export const EMPTY_ORG_SPENDING_REPORT: StorybookSpendingReport = {
  kpiTotals: {
    costCents: "0",
    totalTokens: "0",
    durationSeconds: "0",
    hostedCostCents: "0",
    byokCostCents: "0",
  },
  explorerTotals: {
    costCents: "0",
    totalTokens: "0",
    durationSeconds: "0",
  },
  series: [],
  seriesKeys: [],
  breakdown: [],
  credit: {
    remainingCreditCents: "5000",
    grantTotalCents: "5000",
    superplaneGrantCents: "5000",
    purchasedCreditCents: "0",
    hostedBilledCents: "0",
    remainingCreditWarning: false,
    billingEnabled: false,
    hasBillingCustomer: false,
  },
  catalogs: {
    workspaces: [],
    users: [],
    models: [],
    machines: [],
  },
};

import type { OrganizationsDescribeOrganizationSpendingReportResponse } from "@/api-client";

import type { SpendingCreditSnapshot } from "./spendingRedesignMocks";
import type {
  SpendingCatalogItem,
  SpendingCatalogs,
  SpendingDateRange,
  SpendingReport,
  SpendingTotals,
} from "./spendingRedesignLib";

function metric(value?: string | number | null): number {
  if (value === undefined || value === null || value === "") {
    return 0;
  }
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function mapSpendingCreditSnapshot(
  credit: OrganizationsDescribeOrganizationSpendingReportResponse["credit"],
): SpendingCreditSnapshot {
  return {
    remainingCreditCents: metric(credit?.remainingCreditCents),
    grantTotalCents: metric(credit?.grantTotalCents),
    superplaneGrantCents: metric(credit?.superplaneGrantCents),
    purchasedCreditCents: metric(credit?.purchasedCreditCents),
    hostedBilledCents: metric(credit?.hostedBilledCents),
    remainingCreditWarning: credit?.remainingCreditWarning === true,
    billingEnabled: credit?.billingEnabled === true,
    hasBillingCustomer: credit?.hasBillingCustomer === true,
  };
}

export function mapSpendingCatalogs(
  catalogs: OrganizationsDescribeOrganizationSpendingReportResponse["catalogs"],
): SpendingCatalogs {
  const mapItems = (items?: Array<{ id?: string; label?: string }>): SpendingCatalogItem[] =>
    (items ?? [])
      .filter((item): item is { id: string; label: string } => Boolean(item.id))
      .map((item) => ({ id: item.id, label: item.label || item.id }));

  return {
    workspaces: mapItems(catalogs?.workspaces),
    users: mapItems(catalogs?.users),
    models: mapItems(catalogs?.models),
    machines: mapItems(catalogs?.machines),
  };
}

export function mapSpendingKpiTotals(
  totals: OrganizationsDescribeOrganizationSpendingReportResponse["kpiTotals"],
): SpendingTotals {
  return {
    costCents: metric(totals?.costCents),
    tokens: metric(totals?.totalTokens),
    durationSeconds: metric(totals?.durationSeconds),
    hostedCostCents: metric(totals?.hostedCostCents),
    byokCostCents: metric(totals?.byokCostCents),
  };
}

export function mapSpendingExplorerReport(
  response: OrganizationsDescribeOrganizationSpendingReportResponse,
  range: SpendingDateRange,
): SpendingReport {
  const totals = response.explorerTotals;
  const seriesKeys = (response.seriesKeys ?? [])
    .filter((item): item is { id: string; label: string } => Boolean(item.id))
    .map((item) => ({ id: item.id, label: item.label || item.id }));

  return {
    range,
    totals: {
      costCents: metric(totals?.costCents),
      tokens: metric(totals?.totalTokens),
      durationSeconds: metric(totals?.durationSeconds),
      hostedCostCents: 0,
      byokCostCents: 0,
    },
    series: (response.series ?? []).map((point) => ({
      key: point.key ?? "",
      label: point.label ?? "",
      totalCents: metric(point.totalCents),
      values: Object.fromEntries(
        (point.values ?? [])
          .filter((value) => value.seriesId)
          .map((value) => [value.seriesId as string, metric(value.costCents)]),
      ),
    })),
    seriesKeys,
    breakdown: (response.breakdown ?? []).map((row) => ({
      id: row.id ?? "",
      label: row.label ?? row.id ?? "",
      tokens: metric(row.totalTokens),
      durationSeconds: metric(row.durationSeconds),
      costCents: metric(row.costCents),
      share: row.share ?? 0,
    })),
  };
}

import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { organizationsDescribeOrganizationSpendingReport } from "@/api-client";
import type { OrganizationsDescribeOrganizationSpendingReportResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type {
  SpendingBreakdown,
  SpendingDateRange,
  SpendingFilters,
} from "@/pages/factories/pages/organizationSettings/spending-redesign/spendingRedesignLib";
import { spendingTimeGrainForRange } from "@/pages/factories/pages/organizationSettings/spending-redesign/spendingRedesignLib";

export interface OrganizationSpendingReportQuery {
  organizationId: string;
  range: SpendingDateRange;
  usageKind: "model" | "compute";
  filters: SpendingFilters;
  groupBy: SpendingBreakdown;
}

function spendingReportQueryParams(query: OrganizationSpendingReportQuery) {
  const { filters, groupBy, range, usageKind } = query;
  return {
    startTime: range.start.toISOString(),
    endTime: range.end.toISOString(),
    factoryId: filters.workspaceId || undefined,
    model: filters.model || undefined,
    machineType: filters.machineType || undefined,
    taskOwnerId: filters.userId || undefined,
    groupBy,
    timeGrain: spendingTimeGrainForRange(range),
    usageKind,
  };
}

export function useOrganizationSpendingReport(query: OrganizationSpendingReportQuery, enabled = true) {
  const { organizationId } = query;

  return useQuery({
    queryKey: [
      "organizations",
      organizationId,
      "spending-report",
      query.range.start.toISOString(),
      query.range.end.toISOString(),
      query.usageKind,
      query.filters,
      query.groupBy,
    ] as const,
    queryFn: async (): Promise<OrganizationsDescribeOrganizationSpendingReportResponse> => {
      const response = await organizationsDescribeOrganizationSpendingReport(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          query: spendingReportQueryParams(query),
        }),
      );
      return response.data ?? {};
    },
    enabled: enabled && Boolean(organizationId),
    staleTime: 30 * 1000,
    // Keep the previously loaded report visible while a new range, filter, or
    // grouping is fetching. Without this, the query key reset reports
    // isLoading again and the page swaps every panel for a loading message.
    placeholderData: keepPreviousData,
  });
}

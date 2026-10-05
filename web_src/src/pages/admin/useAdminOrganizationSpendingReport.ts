import { keepPreviousData, useQuery } from "@tanstack/react-query";

import type { OrganizationsDescribeOrganizationSpendingReportResponse } from "@/api-client";
import { spendingReportQueryParams, type OrganizationSpendingReportQuery } from "@/hooks/useOrganizationSpendingReport";

function spendingReportSearch(query: OrganizationSpendingReportQuery): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(spendingReportQueryParams(query))) {
    if (value) {
      params.set(key, value);
    }
  }
  return params.toString();
}

export function useAdminOrganizationSpendingReport(query: OrganizationSpendingReportQuery) {
  const { organizationId } = query;

  return useQuery({
    queryKey: [
      "admin",
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
      const search = spendingReportSearch(query);
      const response = await fetch(`/admin/api/organizations/${organizationId}/spending-report?${search}`, {
        credentials: "include",
      });
      if (!response.ok) {
        throw new Error("Unable to load spending.");
      }
      return (await response.json()) as OrganizationsDescribeOrganizationSpendingReportResponse;
    },
    enabled: Boolean(organizationId),
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });
}

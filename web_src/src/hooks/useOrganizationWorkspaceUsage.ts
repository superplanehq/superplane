import { useQuery } from "@tanstack/react-query";

import { organizationsDescribeOrganizationWorkspaceUsage } from "@/api-client";
import type { OrganizationsDescribeOrganizationWorkspaceUsageResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function useOrganizationWorkspaceUsage(organizationId: string, periodDays = 30) {
  return useQuery({
    queryKey: ["organizations", organizationId, "workspace-usage", periodDays] as const,
    queryFn: async (): Promise<OrganizationsDescribeOrganizationWorkspaceUsageResponse> => {
      const response = await organizationsDescribeOrganizationWorkspaceUsage(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          query: { periodDays },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId),
    staleTime: 30 * 1000,
  });
}

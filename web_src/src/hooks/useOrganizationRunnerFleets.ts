import { useQuery } from "@tanstack/react-query";

import { organizationsListOrganizationRunnerFleets } from "@/api-client";
import type { OrganizationsListOrganizationRunnerFleetsResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function organizationRunnerFleetsQueryKey(organizationId: string) {
  return ["organizations", organizationId, "runner-fleets"] as const;
}

export function useOrganizationRunnerFleets(organizationId: string) {
  return useQuery({
    queryKey: organizationRunnerFleetsQueryKey(organizationId),
    queryFn: async (): Promise<OrganizationsListOrganizationRunnerFleetsResponse> => {
      const response = await organizationsListOrganizationRunnerFleets(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId),
    staleTime: 30 * 1000,
  });
}

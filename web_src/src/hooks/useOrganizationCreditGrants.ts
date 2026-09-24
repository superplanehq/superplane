import { useQuery } from "@tanstack/react-query";

import { organizationsListOrganizationCreditGrants } from "@/api-client";
import type { OrganizationsListOrganizationCreditGrantsResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function organizationCreditGrantsQueryKey(organizationId: string) {
  return ["organizations", organizationId, "credit-grants"] as const;
}

export function useOrganizationCreditGrants(organizationId: string) {
  return useQuery({
    queryKey: organizationCreditGrantsQueryKey(organizationId),
    queryFn: async (): Promise<OrganizationsListOrganizationCreditGrantsResponse> => {
      const response = await organizationsListOrganizationCreditGrants(
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

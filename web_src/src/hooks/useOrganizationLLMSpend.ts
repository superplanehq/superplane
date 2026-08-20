import { useQuery } from "@tanstack/react-query";

import { organizationsDescribeOrganizationLlmSpend } from "@/api-client";
import type { OrganizationsDescribeOrganizationLlmSpendResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function useOrganizationLLMSpend(organizationId: string, periodDays = 30) {
  return useQuery({
    queryKey: ["organizations", organizationId, "llm-spend", periodDays] as const,
    queryFn: async (): Promise<OrganizationsDescribeOrganizationLlmSpendResponse> => {
      const response = await organizationsDescribeOrganizationLlmSpend(
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

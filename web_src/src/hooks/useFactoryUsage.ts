import { useQuery } from "@tanstack/react-query";

import { factoriesDescribeFactoryUsage } from "@/api-client";
import type { FactoriesDescribeFactoryUsageResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { factoryQueryKeys } from "./useFactoryData";

export function useFactoryUsage(organizationId: string, factoryId: string, periodDays = 30) {
  return useQuery({
    queryKey: [...factoryQueryKeys.detail(organizationId, factoryId), "usage", periodDays] as const,
    queryFn: async (): Promise<FactoriesDescribeFactoryUsageResponse> => {
      const response = await factoriesDescribeFactoryUsage(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: { periodDays },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && factoryId),
    staleTime: 30 * 1000,
  });
}

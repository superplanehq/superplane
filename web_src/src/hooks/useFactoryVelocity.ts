import { useQuery } from "@tanstack/react-query";

import { factoriesDescribeFactoryVelocity } from "@/api-client";
import type { FactoriesDescribeFactoryVelocityResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { factoryQueryKeys } from "./useFactoryData";

export interface FactoryVelocityParams {
  periodDays: number;
  integrationId?: string;
  repository?: string;
}

export function useFactoryVelocity(organizationId: string, factoryId: string, params: FactoryVelocityParams) {
  return useQuery({
    queryKey: factoryQueryKeys.velocity(
      organizationId,
      factoryId,
      params.periodDays,
      params.integrationId ?? "",
      params.repository ?? "",
    ),
    queryFn: async (): Promise<FactoriesDescribeFactoryVelocityResponse> => {
      const query: Record<string, string | number> = { periodDays: params.periodDays };
      if (params.integrationId) query.integrationId = params.integrationId;
      if (params.repository) query.repository = params.repository;

      const response = await factoriesDescribeFactoryVelocity(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query,
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && factoryId),
    staleTime: 60 * 1000,
  });
}

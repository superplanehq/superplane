import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { factoriesListFactoryWorkOrderRunUsage } from "@/api-client";
import type { FactoriesListFactoryWorkOrderRunUsageResponse } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { factoryQueryKeys } from "./useFactoryData";

export const WORK_ORDER_RUN_USAGE_PAGE_SIZE = 50;

export interface FactoryWorkOrderRunUsageQuery {
  organizationId: string;
  factoryId: string;
  startTime: string;
  endTime: string;
  offset: number;
  limit?: number;
}

export function useFactoryWorkOrderRunUsage(query: FactoryWorkOrderRunUsageQuery, enabled = true) {
  const { organizationId, factoryId, startTime, endTime, offset } = query;
  const limit = query.limit ?? WORK_ORDER_RUN_USAGE_PAGE_SIZE;

  return useQuery({
    queryKey: [
      ...factoryQueryKeys.detail(organizationId, factoryId),
      "usage-history",
      startTime,
      endTime,
      limit,
      offset,
    ] as const,
    queryFn: async (): Promise<FactoriesListFactoryWorkOrderRunUsageResponse> => {
      const response = await factoriesListFactoryWorkOrderRunUsage(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          query: { startTime, endTime, limit, offset },
        }),
      );
      return response.data ?? {};
    },
    enabled: enabled && Boolean(organizationId && factoryId),
    staleTime: 30 * 1000,
    placeholderData: keepPreviousData,
  });
}

import { factoriesListWorkOrderChecks } from "@/api-client";
import type { FactoriesWorkOrderCheck } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQueries, useQuery } from "@tanstack/react-query";

import { factoryQueryKeys } from "./useFactoryData";

/**
 * Live analysis writes confidence without a factory websocket. Poll while
 * the Backlog run is in flight so the board card updates after the popup
 * closes.
 */
export const ANALYZING_WORK_ORDER_CHECKS_POLL_MS = 1500;

export function useWorkOrderChecks(
  organizationId: string,
  factoryId: string,
  orderId: string,
  options?: { enabled?: boolean; refetchInterval?: number | false },
) {
  return useQuery(workOrderChecksQuery(organizationId, factoryId, orderId, options));
}

/** Prefetch the same check cache LineBoardOrderCard reads. */
export function useWorkOrderChecksForOrders(
  organizationId: string,
  factoryId: string,
  orderIds: readonly string[],
  enabled = true,
) {
  return useQueries({
    queries: orderIds.map((orderId) => workOrderChecksQuery(organizationId, factoryId, orderId, { enabled })),
  });
}

function workOrderChecksQuery(
  organizationId: string,
  factoryId: string,
  orderId: string,
  options?: { enabled?: boolean; refetchInterval?: number | false },
) {
  return {
    queryKey: factoryQueryKeys.workOrderChecks(organizationId, factoryId, orderId),
    queryFn: async (): Promise<FactoriesWorkOrderCheck[]> => {
      const response = await factoriesListWorkOrderChecks(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId },
        }),
      );
      return response.data?.checks ?? [];
    },
    enabled: Boolean(organizationId && factoryId && orderId) && (options?.enabled ?? true),
    refetchInterval: options?.refetchInterval,
  };
}

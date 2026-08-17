import { factoriesListLineMetrics } from "@/api-client";
import type { FactoriesLineMetrics } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";
import { factoryQueryKeys } from "./useFactoryData";

/**
 * Trailing-window (default 30 days) success/rework/cost/throughput metrics,
 * one entry per line that had at least one closed work order in the window.
 * A line with no data simply won't be a key in the returned record — callers
 * should treat a missing entry as "no data" (render dashes), not as zeroes.
 */
export function useFactoryLineMetrics(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryQueryKeys.lineMetrics(organizationId, factoryId),
    queryFn: async (): Promise<Record<string, FactoriesLineMetrics>> => {
      const response = await factoriesListLineMetrics(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
        }),
      );
      const metrics = response.data?.metrics ?? [];
      return Object.fromEntries(metrics.filter((entry) => entry.lineId).map((entry) => [entry.lineId, entry]));
    },
    enabled: Boolean(organizationId && factoryId),
  });
}

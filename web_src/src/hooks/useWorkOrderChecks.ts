import { factoriesListWorkOrderChecks } from "@/api-client";
import type { FactoriesWorkOrderCheck } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";

import { factoryQueryKeys } from "./useFactoryData";

export function useWorkOrderChecks(organizationId: string, factoryId: string, orderId: string) {
  return useQuery({
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
    enabled: Boolean(organizationId && factoryId && orderId),
  });
}

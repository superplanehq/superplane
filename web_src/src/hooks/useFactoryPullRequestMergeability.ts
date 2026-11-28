import { factoriesDescribeFactoryPullRequestMergeability } from "@/api-client";
import type { FactoriesFactoryPullRequestMergeability } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useQuery } from "@tanstack/react-query";

import { factoryQueryKeys } from "./useFactoryData";

export function useFactoryPullRequestMergeability(
  organizationId: string,
  factoryId: string,
  pullRequestId: string,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: factoryQueryKeys.pullRequestMergeability(organizationId, factoryId, pullRequestId),
    queryFn: async (): Promise<FactoriesFactoryPullRequestMergeability> => {
      const response = await factoriesDescribeFactoryPullRequestMergeability(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, prId: pullRequestId },
        }),
      );
      return response.data?.mergeability ?? {};
    },
    enabled: Boolean(organizationId && factoryId && pullRequestId) && (options?.enabled ?? true),
  });
}

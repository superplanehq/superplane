import { factoriesDescribeFactoryPullRequestMergeability, factoriesMergeFactoryPullRequest } from "@/api-client";
import type { FactoriesFactoryPullRequestMergeability, FactoryPullRequestMergeabilityMergeMethod } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { factoryQueryKeys } from "./useFactoryData";
import { invalidateFactoryWorkOrderQueries } from "./useFactoryWebsocket";

export function factoryPullRequestMergeabilityKey(organizationId: string, factoryId: string, pullRequestId: string) {
  return factoryQueryKeys.pullRequestMergeability(organizationId, factoryId, pullRequestId);
}

export function useFactoryPullRequestMergeability(
  organizationId: string,
  factoryId: string,
  pullRequestId: string,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: factoryPullRequestMergeabilityKey(organizationId, factoryId, pullRequestId),
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

export function useMergeFactoryPullRequest(organizationId: string, factoryId: string, orderId?: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      pullRequestId: string;
      mergeMethod: FactoryPullRequestMergeabilityMergeMethod;
      expectedHeadSha: string;
    }) => {
      const response = await factoriesMergeFactoryPullRequest(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, prId: input.pullRequestId },
          body: {
            mergeMethod: input.mergeMethod,
            expectedHeadSha: input.expectedHeadSha,
          },
        }),
      );
      if (!response.data?.pullRequest) {
        throw new Error("Failed to merge pull request");
      }
      return response.data.pullRequest;
    },
    onSuccess: (_pullRequest, input) => {
      void queryClient.invalidateQueries({
        queryKey: factoryPullRequestMergeabilityKey(organizationId, factoryId, input.pullRequestId),
      });
      invalidateFactoryWorkOrderQueries(queryClient, organizationId, factoryId, orderId);
    },
  });
}

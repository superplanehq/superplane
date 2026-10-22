import { factoriesContinueFactoryPullRequestFixes } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { invalidateFactoryWorkOrderQueries } from "./useFactoryWebsocket";

export type ContinueFactoryPullRequestFixesInput = {
  pullRequestId: string;
  runId: string;
};

/**
 * Resume automatic check fixes that the attempt limit paused. The work order
 * queries carry the pull request activities, so refreshing them also refreshes
 * the pull request feedback log.
 */
export function useContinueFactoryPullRequestFixes(organizationId: string, factoryId: string, orderId?: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: ContinueFactoryPullRequestFixesInput) => {
      const response = await factoriesContinueFactoryPullRequestFixes(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, prId: input.pullRequestId },
          body: { runId: input.runId },
        }),
      );
      return response.data?.eventId;
    },
    onSuccess: () => {
      invalidateFactoryWorkOrderQueries(queryClient, organizationId, factoryId, orderId);
    },
  });
}

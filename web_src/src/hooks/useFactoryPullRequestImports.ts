import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { factoriesImportFactoryPullRequest, factoriesSearchFactoryPullRequestImportItems } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useCodexActionGuard } from "@/pages/factories/codex/codexActionContext";
import { factoryQueryKeys } from "./useFactoryData";
import { applyWorkOrderToListCaches } from "./workOrderListCache";

const candidatesKey = (organizationId: string, factoryId: string) =>
  ["factories", organizationId, factoryId, "pull-request-imports"] as const;

export function useSearchFactoryPullRequestImports(
  organizationId: string,
  factoryId: string,
  query: string,
  limit: number,
  enabled: boolean,
) {
  return useQuery({
    queryKey: [...candidatesKey(organizationId, factoryId), query, limit],
    enabled,
    queryFn: async () => {
      const response = await factoriesSearchFactoryPullRequestImportItems(
        withOrganizationHeader({ organizationId, path: { factoryId }, query: { query, limit } }),
      );
      return response.data?.items ?? [];
    },
    placeholderData: (previous, previousQuery) => {
      const key = previousQuery?.queryKey;
      if (key?.[1] !== organizationId || key?.[2] !== factoryId || key?.[4] !== query) return undefined;
      return previous;
    },
  });
}

export function useImportFactoryPullRequest(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  const guard = useCodexActionGuard(organizationId, factoryId);
  return useMutation({
    mutationFn: (input: { lineId: string; pullRequestUrl: string }) =>
      guard(async () => {
        const response = await factoriesImportFactoryPullRequest(
          withOrganizationHeader({ organizationId, path: { factoryId }, body: input }),
        );
        if (!response.data?.order) throw new Error("SuperPlane could not import the pull request.");
        return response.data.order;
      }),
    onSuccess: (order) => {
      applyWorkOrderToListCaches(queryClient, organizationId, factoryId, order.id ?? "", order);
      if (order.id)
        queryClient.setQueryData(factoryQueryKeys.workOrderDetail(organizationId, factoryId, order.id), order);
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrdersPagePrefix(organizationId, factoryId),
      });
      void queryClient.invalidateQueries({ queryKey: candidatesKey(organizationId, factoryId) });
    },
  });
}

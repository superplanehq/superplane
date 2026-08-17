import { factoriesAddWorkOrderReaction, factoriesRemoveWorkOrderReaction } from "@/api-client";
import type { FactoriesWorkOrder } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { factoryQueryKeys } from "./useFactoryData";

/**
 * Merges a freshly-fetched reaction rollup into the cached work order, so
 * the actor sees their own toggle applied immediately without waiting for a
 * refetch — other clients pick it up via the factory websocket
 * (`order.reaction.updated`), which invalidates this same query key.
 */
function setCachedWorkOrderReactions(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
  orderId: string,
  reactions: FactoriesWorkOrder["reactions"],
) {
  const queryKey = factoryQueryKeys.workOrderDetail(organizationId, factoryId, orderId);
  queryClient.setQueryData<FactoriesWorkOrder | undefined>(queryKey, (current) =>
    current ? { ...current, reactions } : current,
  );
}

export function useAddWorkOrderReaction(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; content: string }) => {
      const response = await factoriesAddWorkOrderReaction(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId },
          body: { content: input.content },
        }),
      );
      return response.data?.reactions ?? [];
    },
    onSuccess: (reactions, variables) => {
      setCachedWorkOrderReactions(queryClient, organizationId, factoryId, variables.orderId, reactions);
    },
  });
}

export function useRemoveWorkOrderReaction(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; content: string }) => {
      const response = await factoriesRemoveWorkOrderReaction(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId, content: input.content },
        }),
      );
      return response.data?.reactions ?? [];
    },
    onSuccess: (reactions, variables) => {
      setCachedWorkOrderReactions(queryClient, organizationId, factoryId, variables.orderId, reactions);
    },
  });
}

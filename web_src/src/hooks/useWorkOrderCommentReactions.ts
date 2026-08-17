import {
  factoriesAddWorkOrderCommentReaction,
  factoriesRemoveWorkOrderCommentReaction,
  type FactoriesListWorkOrderEventsResponse,
  type FactoriesWorkOrderCommentReactionSummary,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { type InfiniteData, type QueryClient, useMutation, useQueryClient } from "@tanstack/react-query";
import { factoryQueryKeys } from "./useFactoryData";

interface CommentReactionCachePatch {
  organizationId: string;
  factoryId: string;
  orderId: string;
  commentId: string;
  reactions: FactoriesWorkOrderCommentReactionSummary[];
}

// Both reaction mutations return the comment's freshly-recomputed
// summary, so the local viewer's UI updates from the response directly
// instead of waiting on a refetch. Other viewers still pick up the
// change via the `order.comment.reaction.updated` websocket message,
// which invalidates this same query key (see useFactoryWebsocket).
function patchCommentReactionsInEventsCache(queryClient: QueryClient, patch: CommentReactionCachePatch) {
  const { organizationId, factoryId, orderId, commentId, reactions } = patch;

  queryClient.setQueryData<InfiniteData<FactoriesListWorkOrderEventsResponse>>(
    factoryQueryKeys.workOrderEvents(organizationId, factoryId, orderId),
    (data) => {
      if (!data) return data;

      return {
        ...data,
        pages: data.pages.map((page) => ({
          ...page,
          events: (page.events ?? []).map((event) =>
            event.id === commentId ? { ...event, event: { ...(event.event ?? {}), reactions } } : event,
          ),
        })),
      };
    },
  );
}

export function useAddWorkOrderCommentReaction(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; commentId: string; emoji: string }) => {
      const response = await factoriesAddWorkOrderCommentReaction(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId, commentId: input.commentId },
          body: { emoji: input.emoji },
        }),
      );
      return response.data?.reactions ?? [];
    },
    onSuccess: (reactions, variables) => {
      patchCommentReactionsInEventsCache(queryClient, {
        organizationId,
        factoryId,
        orderId: variables.orderId,
        commentId: variables.commentId,
        reactions,
      });
    },
  });
}

export function useRemoveWorkOrderCommentReaction(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { orderId: string; commentId: string; emoji: string }) => {
      const response = await factoriesRemoveWorkOrderCommentReaction(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, orderId: input.orderId, commentId: input.commentId, emoji: input.emoji },
        }),
      );
      return response.data?.reactions ?? [];
    },
    onSuccess: (reactions, variables) => {
      patchCommentReactionsInEventsCache(queryClient, {
        organizationId,
        factoryId,
        orderId: variables.orderId,
        commentId: variables.commentId,
        reactions,
      });
    },
  });
}

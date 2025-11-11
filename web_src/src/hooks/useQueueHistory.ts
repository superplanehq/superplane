import { useCallback, useMemo } from "react";
import { useInfiniteNodeQueueItems } from "./useWorkflowData";
import { SidebarEvent } from "@/ui/CanvasPage";
import { ComponentsNode, WorkflowsListNodeQueueItemsResponse } from "@/api-client";
import { mapQueueItemsToSidebarEvents } from "@/pages/workflowv2/utils";

interface UseQueueHistoryProps {
  workflowId: string;
  nodeId: string;
  allNodes: ComponentsNode[];
  enabled: boolean;
}

export const useQueueHistory = ({ workflowId, nodeId, allNodes, enabled }: UseQueueHistoryProps) => {
  const queueItemsQuery = useInfiniteNodeQueueItems(workflowId, nodeId, enabled);

  const getAllHistoryEvents = useCallback((): SidebarEvent[] => {
    if (!enabled) return [];

    const node = allNodes.find((n) => n.id === nodeId);
    if (!node) return [];

    const allQueueItems =
      queueItemsQuery.data?.pages.flatMap((page) => (page as WorkflowsListNodeQueueItemsResponse)?.items || []) || [];
    return mapQueueItemsToSidebarEvents(allQueueItems, allNodes);
  }, [enabled, allNodes, nodeId, queueItemsQuery.data]);

  const handleLoadMore = useCallback(() => {
    queueItemsQuery.fetchNextPage();
  }, [queueItemsQuery]);

  const hasMoreHistory = useMemo(() => queueItemsQuery.hasNextPage, [queueItemsQuery.hasNextPage]);
  const isLoadingMore = useMemo(() => queueItemsQuery.isFetchingNextPage, [queueItemsQuery.isFetchingNextPage]);

  return {
    getAllHistoryEvents,
    handleLoadMore,
    hasMoreHistory: hasMoreHistory || false,
    isLoadingMore: isLoadingMore || false,
  };
};

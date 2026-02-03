import { useCallback, useMemo } from "react";
import { useInfiniteNodeQueueItems } from "./useCanvasData";
import { ComponentsNode, CanvasesListNodeQueueItemsResponse } from "@/api-client";
import { mapQueueItemsToSidebarEvents } from "@/pages/workflowv2/utils";
import { SidebarEvent } from "@/ui/componentSidebar/types";

interface UseQueueHistoryProps {
  canvasId: string;
  nodeId: string;
  allNodes: ComponentsNode[];
  enabled: boolean;
}

export const useQueueHistory = ({ canvasId, nodeId, allNodes, enabled }: UseQueueHistoryProps) => {
  const queueItemsQuery = useInfiniteNodeQueueItems(canvasId, nodeId, enabled);

  const getAllHistoryEvents = useCallback((): SidebarEvent[] => {
    if (!enabled) return [];

    const node = allNodes.find((n) => n.id === nodeId);
    if (!node) return [];

    const allQueueItems =
      queueItemsQuery.data?.pages.flatMap((page) => (page as CanvasesListNodeQueueItemsResponse)?.items || []) || [];
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

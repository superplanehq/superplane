import { useCallback, useMemo } from "react";
import { useInfiniteNodeEvents, useInfiniteNodeExecutions } from "./useWorkflowData";
import { SidebarEvent } from "@/ui/CanvasPage";
import { ComponentsNode, WorkflowsListNodeEventsResponse, WorkflowsListNodeExecutionsResponse } from "@/api-client";
import { mapTriggerEventsToSidebarEvents, mapExecutionsToSidebarEvents } from "@/pages/workflowv2/utils";

interface UseNodeHistoryProps {
  workflowId: string;
  nodeId: string;
  nodeType: string;
  allNodes: ComponentsNode[];
  enabled: boolean;
}

export const useNodeHistory = ({ workflowId, nodeId, nodeType, allNodes, enabled }: UseNodeHistoryProps) => {
  // For trigger nodes, use events; for other nodes, use executions
  const isTriggerNode = nodeType === "TYPE_TRIGGER";

  const eventsQuery = useInfiniteNodeEvents(workflowId, nodeId, enabled && isTriggerNode);
  const executionsQuery = useInfiniteNodeExecutions(workflowId, nodeId, enabled && !isTriggerNode);

  const getAllHistoryEvents = useCallback((): SidebarEvent[] => {
    if (!enabled) return [];

    const node = allNodes.find((n) => n.id === nodeId);
    if (!node) return [];

    if (isTriggerNode) {
      const allEvents = eventsQuery.data?.pages.flatMap((page) => (page as WorkflowsListNodeEventsResponse)?.events || []) || [];
      return mapTriggerEventsToSidebarEvents(allEvents, node);
    } else {
      const allExecutions = executionsQuery.data?.pages.flatMap((page) => (page as WorkflowsListNodeExecutionsResponse)?.executions || []) || [];
      return mapExecutionsToSidebarEvents(allExecutions, allNodes);
    }
  }, [enabled, allNodes, nodeId, isTriggerNode, eventsQuery.data, executionsQuery.data]);

  const handleLoadMore = useCallback(() => {
    if (isTriggerNode) {
      eventsQuery.fetchNextPage();
    } else {
      executionsQuery.fetchNextPage();
    }
  }, [isTriggerNode, eventsQuery, executionsQuery]);

  const hasMoreHistory = useMemo(() => isTriggerNode ? eventsQuery.hasNextPage : executionsQuery.hasNextPage, [isTriggerNode, eventsQuery.hasNextPage, executionsQuery.hasNextPage]);
  const isLoadingMore = useMemo(() => isTriggerNode ? eventsQuery.isFetchingNextPage : executionsQuery.isFetchingNextPage, [isTriggerNode, eventsQuery.isFetchingNextPage, executionsQuery.isFetchingNextPage]);

  return {
    getAllHistoryEvents,
    handleLoadMore,
    hasMoreHistory: hasMoreHistory || false,
    isLoadingMore: isLoadingMore || false,
  };
};
import { useCallback, useMemo } from "react";
import { useInfiniteNodeEvents, useInfiniteNodeExecutions } from "./useWorkflowData";
import { SidebarEvent } from "@/ui/componentSidebar/types";
import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsListNodeEventsResponse,
  WorkflowsListNodeExecutionsResponse,
} from "@/api-client";
import { mapTriggerEventsToSidebarEvents, mapExecutionsToSidebarEvents } from "@/pages/workflowv2/utils";
import { QueryClient } from "@tanstack/react-query";
import { getComponentAdditionalDataBuilder } from "@/pages/workflowv2/mappers";

interface UseNodeHistoryProps {
  workflowId: string;
  organizationId: string;
  components: ComponentsComponent[];
  nodeId: string;
  nodeType: string;
  allNodes: ComponentsNode[];
  enabled: boolean;
  queryClient: QueryClient;
}

export const useNodeHistory = ({
  workflowId,
  nodeId,
  nodeType,
  allNodes,
  enabled,
  organizationId,
  queryClient,
  components,
}: UseNodeHistoryProps) => {
  // For trigger nodes, use events; for other nodes, use executions
  const isTriggerNode = nodeType === "TYPE_TRIGGER";

  const eventsQuery = useInfiniteNodeEvents(workflowId, nodeId, enabled && isTriggerNode);
  const executionsQuery = useInfiniteNodeExecutions(workflowId, nodeId, enabled && !isTriggerNode);

  const getAllHistoryEvents = useCallback((): SidebarEvent[] => {
    if (!enabled) return [];

    const node = allNodes.find((n) => n.id === nodeId);
    if (!node) return [];

    if (isTriggerNode) {
      const allEvents =
        eventsQuery.data?.pages.flatMap((page) => (page as WorkflowsListNodeEventsResponse)?.events || []) || [];
      return mapTriggerEventsToSidebarEvents(allEvents, node);
    } else {
      const allExecutions =
        executionsQuery.data?.pages.flatMap(
          (page) => (page as WorkflowsListNodeExecutionsResponse)?.executions || [],
        ) || [];

      const componentDef = components.find((c) => c.name === node.component?.name);

      const additionalData = getComponentAdditionalDataBuilder(componentDef?.name || "")?.buildAdditionalData(
        allNodes,
        node,
        componentDef!,
        allExecutions,
        workflowId || "",
        queryClient,
        organizationId || "",
      );

      return mapExecutionsToSidebarEvents(allExecutions, allNodes, undefined, additionalData);
    }
  }, [
    enabled,
    allNodes,
    nodeId,
    isTriggerNode,
    eventsQuery.data,
    executionsQuery.data,
    components,
    organizationId,
    queryClient,
    workflowId,
  ]);

  const handleLoadMore = useCallback(() => {
    if (isTriggerNode) {
      eventsQuery.fetchNextPage();
    } else {
      executionsQuery.fetchNextPage();
    }
  }, [isTriggerNode, eventsQuery, executionsQuery]);

  const hasMoreHistory = useMemo(
    () => (isTriggerNode ? eventsQuery.hasNextPage : executionsQuery.hasNextPage),
    [isTriggerNode, eventsQuery.hasNextPage, executionsQuery.hasNextPage],
  );
  const isLoadingMore = useMemo(
    () => (isTriggerNode ? eventsQuery.isFetchingNextPage : executionsQuery.isFetchingNextPage),
    [isTriggerNode, eventsQuery.isFetchingNextPage, executionsQuery.isFetchingNextPage],
  );

  return {
    getAllHistoryEvents,
    handleLoadMore,
    hasMoreHistory: hasMoreHistory || false,
    isLoadingMore: isLoadingMore || false,
  };
};

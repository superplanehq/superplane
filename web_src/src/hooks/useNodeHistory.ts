import { useCallback, useMemo } from "react";
import { useInfiniteNodeEvents, useInfiniteNodeExecutions } from "./useCanvasData";
import type { SidebarEvent } from "@/ui/componentSidebar/types";
import type {
  ComponentsComponent,
  ComponentsNode,
  CanvasesListNodeEventsResponse,
  CanvasesListNodeExecutionsResponse,
} from "@/api-client";
import { mapTriggerEventsToSidebarEvents, mapExecutionsToSidebarEvents } from "@/pages/workflowv2/utils";
import type { QueryClient } from "@tanstack/react-query";
import { useMe } from "./useMe";

interface UseNodeHistoryProps {
  canvasId: string;
  organizationId: string;
  components: ComponentsComponent[];
  nodeId: string;
  nodeType: string;
  allNodes: ComponentsNode[];
  enabled: boolean;
  queryClient: QueryClient;
}

export const useNodeHistory = ({
  canvasId,
  nodeId,
  nodeType,
  allNodes,
  enabled,
  organizationId,
  queryClient,
  components,
}: UseNodeHistoryProps) => {
  const { data: me } = useMe();

  // For trigger nodes, use events; for other nodes, use executions
  const isTriggerNode = nodeType === "TYPE_TRIGGER";

  const eventsQuery = useInfiniteNodeEvents(canvasId, nodeId, enabled && isTriggerNode);
  const executionsQuery = useInfiniteNodeExecutions(canvasId, nodeId, enabled && !isTriggerNode);

  const node = useMemo(() => allNodes.find((n) => n.id === nodeId), [allNodes, nodeId]);
  const componentDef = useMemo(
    () => components.find((c) => c.name === node?.component?.name),
    [components, node?.component?.name],
  );
  const allExecutions = useMemo(() => {
    if (!enabled || isTriggerNode) return [];
    return (
      executionsQuery.data?.pages.flatMap((page) => (page as CanvasesListNodeExecutionsResponse)?.executions || []) ||
      []
    );
  }, [enabled, isTriggerNode, executionsQuery.data]);

  const getAllHistoryEvents = useCallback((): SidebarEvent[] => {
    if (!enabled) return [];

    if (!node) return [];

    if (isTriggerNode) {
      const allEvents =
        eventsQuery.data?.pages.flatMap((page) => (page as CanvasesListNodeEventsResponse)?.events || []) || [];
      return mapTriggerEventsToSidebarEvents(allEvents, node);
    } else {
      return mapExecutionsToSidebarEvents(allExecutions, allNodes, undefined);
    }
  }, [
    enabled,
    node,
    allNodes,
    isTriggerNode,
    eventsQuery.data,
    allExecutions,
    componentDef,
    organizationId,
    queryClient,
    canvasId,
    me,
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

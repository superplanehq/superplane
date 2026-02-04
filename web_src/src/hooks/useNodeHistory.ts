import { useCallback, useMemo } from "react";
import { useInfiniteNodeEvents, useInfiniteNodeExecutions } from "./useCanvasData";
import { SidebarEvent } from "@/ui/componentSidebar/types";
import {
  ComponentsComponent,
  ComponentsNode,
  CanvasesListNodeEventsResponse,
  CanvasesListNodeExecutionsResponse,
} from "@/api-client";
import { mapTriggerEventsToSidebarEvents, mapExecutionsToSidebarEvents, buildComponentDefinition, buildExecutionInfo, buildNodeInfo } from "@/pages/workflowv2/utils";
import { QueryClient } from "@tanstack/react-query";
import { getComponentAdditionalDataBuilder } from "@/pages/workflowv2/mappers";
import { useAccount } from "@/contexts/AccountContext";
import { useApprovalGroupUsersPrefetch } from "@/hooks/useApprovalGroupUsersPrefetch";

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
  const { account } = useAccount();
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
  const approvalGroupNames = useMemo(() => {
    if (!enabled || isTriggerNode || componentDef?.name !== "approval") return [];

    const groupNames = new Set<string>();
    allExecutions.forEach((execution) => {
      const metadata = execution.metadata as { records?: Array<{ type?: string; group?: string }> } | undefined;
      const records = metadata?.records;
      if (!Array.isArray(records)) return;

      records.forEach((record) => {
        if (record.type === "group" && record.group) {
          groupNames.add(record.group);
        }
      });
    });

    return Array.from(groupNames);
  }, [enabled, isTriggerNode, componentDef?.name, allExecutions]);

  useApprovalGroupUsersPrefetch({
    organizationId,
    groupNames: approvalGroupNames,
    enabled: enabled && !isTriggerNode && componentDef?.name === "approval",
  });

  const getAllHistoryEvents = useCallback((): SidebarEvent[] => {
    if (!enabled) return [];

    if (!node) return [];

    if (isTriggerNode) {
      const allEvents =
        eventsQuery.data?.pages.flatMap((page) => (page as CanvasesListNodeEventsResponse)?.events || []) || [];
      return mapTriggerEventsToSidebarEvents(allEvents, node);
    } else {
      const additionalData = getComponentAdditionalDataBuilder(componentDef?.name || "")?.buildAdditionalData(
        {
          nodes: allNodes.map((n) => buildNodeInfo(n)),
          node: buildNodeInfo(node),
          componentDefinition: buildComponentDefinition(componentDef!),
          lastExecutions: allExecutions.map((e) => buildExecutionInfo(e)),
          canvasId: canvasId || "",
          queryClient,
          organizationId: organizationId || "",
          currentUser: account ? { id: account.id, email: account.email } : undefined,
        },
      );

      return mapExecutionsToSidebarEvents(allExecutions, allNodes, undefined, additionalData);
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
    account,
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

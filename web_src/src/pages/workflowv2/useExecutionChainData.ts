import type { QueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import type { CanvasesCanvas, CanvasesCanvasNodeExecution, CanvasesListEventExecutionsResponse } from "@/api-client";
import { eventExecutionsQueryOptions } from "@/hooks/useCanvasData";

export function useExecutionChainData(workflowId: string, queryClient: QueryClient, workflow?: CanvasesCanvas) {
  const loadExecutionChain = useCallback(
    async (
      eventId: string,
      nodeId?: string,
      currentExecution?: Record<string, unknown>,
      forceReload = false,
    ): Promise<CanvasesCanvasNodeExecution[]> => {
      const queryOptions = eventExecutionsQueryOptions(workflowId, eventId);

      let allExecutions: CanvasesCanvasNodeExecution[] = [];

      if (!forceReload) {
        const cachedData = queryClient.getQueryData(queryOptions.queryKey);
        if (cachedData) {
          allExecutions = (cachedData as CanvasesListEventExecutionsResponse)?.executions || [];
        }
      }

      if (allExecutions.length === 0) {
        const options = forceReload ? { ...queryOptions, staleTime: 0 } : queryOptions;
        const data = await queryClient.fetchQuery(options);
        allExecutions = (data as CanvasesListEventExecutionsResponse)?.executions || [];
      }

      // Apply topological filtering - the logic you wanted back!
      if (!allExecutions.length || !workflow || !nodeId) return allExecutions;

      const currentExecutionTime = currentExecution?.createdAt
        ? new Date(currentExecution.createdAt as string).getTime()
        : Date.now();
      const nodesBefore = getNodesBeforeTarget(nodeId, workflow);
      nodesBefore.add(nodeId); // Include current node

      const executionsUpToCurrent = allExecutions.filter((exec) => {
        const execTime = exec.createdAt ? new Date(exec.createdAt).getTime() : 0;
        const isNodeBefore = nodesBefore.has(exec.nodeId || "");
        const isBeforeCurrentTime = execTime <= currentExecutionTime;
        return isNodeBefore && isBeforeCurrentTime;
      });

      // Sort the filtered executions by creation time to get chronological order
      executionsUpToCurrent.sort((a, b) => {
        const timeA = a.createdAt ? new Date(a.createdAt).getTime() : 0;
        const timeB = b.createdAt ? new Date(b.createdAt).getTime() : 0;
        return timeA - timeB;
      });

      return executionsUpToCurrent;
    },
    [workflowId, queryClient, workflow],
  );

  return { loadExecutionChain };
}

// Helper function to build topological path to find all nodes that should execute before the given target node
function getNodesBeforeTarget(targetNodeId: string, workflow: CanvasesCanvas): Set<string> {
  const nodesBefore = new Set<string>();
  if (!workflow?.spec?.edges) return nodesBefore;

  // Build adjacency list for the workflow graph
  const adjacencyList: Record<string, string[]> = {};
  workflow.spec.edges.forEach((edge) => {
    if (!edge.sourceId || !edge.targetId) return;
    if (!adjacencyList[edge.sourceId]) {
      adjacencyList[edge.sourceId] = [];
    }
    adjacencyList[edge.sourceId].push(edge.targetId);
  });

  // DFS to find all nodes that can reach the target
  const visited = new Set<string>();
  const canReachTarget = (nodeId: string): boolean => {
    if (visited.has(nodeId)) return false; // Avoid cycles
    if (nodeId === targetNodeId) return true;

    visited.add(nodeId);
    const neighbors = adjacencyList[nodeId] || [];
    const canReach = neighbors.some((neighbor) => canReachTarget(neighbor));
    visited.delete(nodeId); // Allow revisiting in different paths

    return canReach;
  };

  // Check all nodes to see which ones can reach the target
  const allNodeIds = new Set<string>();
  workflow.spec.edges?.forEach((edge) => {
    if (edge.sourceId) allNodeIds.add(edge.sourceId);
    if (edge.targetId) allNodeIds.add(edge.targetId);
  });
  workflow.spec.nodes?.forEach((node) => {
    if (node.id) allNodeIds.add(node.id);
  });

  allNodeIds.forEach((nodeId) => {
    if (canReachTarget(nodeId)) {
      nodesBefore.add(nodeId);
    }
  });

  return nodesBefore;
}

import { useMemo } from "react";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import {
  buildExecutionTabData,
  buildTriggerTabData,
  buildExecutionChain,
  eventBadgeForExecution,
  eventBadgeForTriggeredTrigger,
  getAdjacentRunNodeId,
  hasObjectValue,
  type RunNodeDetailTabAvailability,
  type RunNodeDetailTabData,
} from "./runNodeDetailModel";

export function useRunNodeDetailPresentation({
  run,
  nodeId,
  workflowNodes,
  executions,
}: {
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes: ComponentsNode[];
  executions: CanvasesCanvasNodeExecution[];
}) {
  const triggerNodeId = run.rootEvent?.nodeId;
  const isTriggerNode = nodeId === triggerNodeId;
  const executionChain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);
  const previousNodeId = useMemo(() => getAdjacentRunNodeId(executionChain, nodeId, "prev"), [executionChain, nodeId]);
  const nextNodeId = useMemo(() => getAdjacentRunNodeId(executionChain, nodeId, "next"), [executionChain, nodeId]);
  const nodeExecution = useMemo(
    () => executions.find((execution) => execution.nodeId === nodeId),
    [executions, nodeId],
  );
  const workflowNode = useMemo(() => workflowNodes.find((node) => node.id === nodeId), [workflowNodes, nodeId]);

  const nodeName = workflowNode?.name || nodeId;
  const createdAt = isTriggerNode ? run.rootEvent?.createdAt : nodeExecution?.createdAt;

  const tabData = useMemo<RunNodeDetailTabData | null>(() => {
    if (isTriggerNode) {
      return buildTriggerTabData(run, workflowNode);
    }

    if (!nodeExecution) return null;
    return buildExecutionTabData(nodeExecution, workflowNode, workflowNodes);
  }, [isTriggerNode, nodeExecution, run, workflowNode, workflowNodes]);

  const hasDetails = !!tabData?.details && Object.keys(tabData.details).length > 0;
  const hasPayload = hasObjectValue(tabData?.payload);
  const hasConfig = hasObjectValue(tabData?.configuration);
  const headerEventBadge = useMemo(() => {
    if (isTriggerNode) return eventBadgeForTriggeredTrigger(workflowNode);
    if (nodeExecution) return eventBadgeForExecution(workflowNode, nodeExecution);
    return null;
  }, [isTriggerNode, nodeExecution, workflowNode]);
  const hasDetailsSection = hasDetails || !!headerEventBadge || !!createdAt;
  const hasAnyTab = hasDetailsSection || hasPayload || hasConfig;
  const tabAvailability = useMemo<RunNodeDetailTabAvailability>(
    () => ({ hasDetailsSection, hasPayload, hasConfig }),
    [hasConfig, hasDetailsSection, hasPayload],
  );

  return {
    workflowNode,
    nodeName,
    createdAt,
    tabData,
    headerEventBadge,
    tabAvailability,
    hasAnyTab,
    hasDetailsSection,
    hasPayload,
    hasConfig,
    isTriggerNode,
    previousNodeId,
    nextNodeId,
  };
}

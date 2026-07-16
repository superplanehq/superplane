import { useMemo } from "react";
import type { ConfigurationField, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { resolveLiveNodePreRunStatus, type LiveNodePreRunStatus } from "@/lib/liveNodePreRunStatus";
import { useNodeExecutionStore } from "@/stores/nodeExecutionStore";

type UseLivePreRunSidebarModelOptions = {
  canvasMode: "live" | "edit";
  formDisabled: boolean;
  isAnnotationNode: boolean;
  isEditing: boolean;
  selectedNodeId: string | null;
  workflowNodes?: ComponentsNode[];
  configurationFields?: ConfigurationField[];
};

export function useLivePreRunSidebarModel({
  canvasMode,
  formDisabled,
  isAnnotationNode,
  isEditing,
  selectedNodeId,
  workflowNodes,
  configurationFields,
}: UseLivePreRunSidebarModelOptions) {
  const nodeExecutionVersion = useNodeExecutionStore((store) => store.version);

  const selectedNodeHasActivity = useMemo(() => {
    if (!selectedNodeId) {
      return false;
    }

    const nodeData = useNodeExecutionStore.getState().getNodeData(selectedNodeId);
    void nodeExecutionVersion;
    return nodeData.executions.length > 0 || nodeData.events.length > 0;
  }, [selectedNodeId, nodeExecutionVersion]);

  const hideRunsTab = isAnnotationNode || (!isEditing && !selectedNodeHasActivity);
  const shouldShowRunsSidebar = canvasMode === "live" && !isAnnotationNode && !hideRunsTab;
  const shouldLoadLiveSidebarData = canvasMode === "live" && !isAnnotationNode;

  const livePreRunStatus = useMemo((): LiveNodePreRunStatus | undefined => {
    if (!formDisabled || !selectedNodeId || !workflowNodes) {
      return undefined;
    }

    const workflowNode = workflowNodes.find((node) => node.id === selectedNodeId);
    if (!workflowNode) {
      return undefined;
    }

    const nodeData = useNodeExecutionStore.getState().getNodeData(selectedNodeId);
    void nodeExecutionVersion;
    return resolveLiveNodePreRunStatus(workflowNode, nodeData, {
      configurationFields,
    });
  }, [configurationFields, formDisabled, nodeExecutionVersion, selectedNodeId, workflowNodes]);

  return {
    hideRunsTab,
    livePreRunStatus,
    selectedNodeHasActivity,
    shouldLoadLiveSidebarData,
    shouldShowRunsSidebar,
  };
}

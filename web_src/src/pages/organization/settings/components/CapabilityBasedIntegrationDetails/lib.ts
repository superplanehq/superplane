import type {
  IntegrationCapabilityStateState,
  IntegrationNodeRef,
  IntegrationsCapabilityDefinition,
} from "@/api-client";

export const DEFAULT_CAPABILITY_STATE: IntegrationCapabilityStateState = "STATE_UNAVAILABLE";

export type DisplayCapability = {
  name: string;
  definition?: IntegrationsCapabilityDefinition;
  state: IntegrationCapabilityStateState;
};

export function getCapabilityLabel(capability: DisplayCapability): string {
  return capability.definition?.label || capability.definition?.name || capability.name || "Unnamed capability";
}

export function getCapabilityDescription(capability: DisplayCapability): string | undefined {
  return capability.definition?.description;
}

export function getCapabilityStatusDotClass(state: IntegrationCapabilityStateState): string {
  if (state === "STATE_ENABLED") return "bg-green-500";
  if (state === "STATE_DISABLED") return "bg-red-500";
  if (state === "STATE_REQUESTED") return "bg-amber-500";
  return "bg-gray-400 dark:bg-gray-500";
}

export const getActiveTabClass = (activeTab?: boolean) => {
  return activeTab
    ? "border-gray-700 text-gray-800 dark:text-blue-400 dark:border-blue-600"
    : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300";
};

export type WorkflowGroup = {
  canvasId: string;
  canvasName: string;
  nodes: Array<{ nodeId: string; nodeName: string }>;
};

export const groupNodeRefsByCanvas = (nodeRefs: IntegrationNodeRef[]): WorkflowGroup[] => {
  if (!nodeRefs) return [];

  const groups = new Map<string, WorkflowGroup>();

  nodeRefs.forEach((nodeRef) => {
    const canvasId = nodeRef.canvasId || "";
    const canvasName = nodeRef.canvasName || canvasId;
    const nodeId = nodeRef.nodeId || "";
    const nodeName = nodeRef.nodeName || nodeId;

    if (!groups.has(canvasId)) {
      groups.set(canvasId, { canvasId, canvasName, nodes: [] });
    }

    groups.get(canvasId)?.nodes.push({ nodeId, nodeName });
  });

  return Array.from(groups.entries()).map(([canvasId, data]) => ({
    canvasId,
    canvasName: data.canvasName,
    nodes: data.nodes,
  }));
};

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

export function getCapabilityStatusLabel(state: IntegrationCapabilityStateState): string {
  switch (state) {
    case "STATE_ENABLED":
      return "Enabled";
    case "STATE_DISABLED":
      return "Disabled";
    case "STATE_REQUESTED":
      return "Requested";
    case "STATE_AVAILABLE":
      return "Available";
    case "STATE_UNAVAILABLE":
      return "Unavailable";
  }
}

/** Outline badge coloring aligned with former status-dot semantics (enabled=green, disabled=red, …). */
export function getCapabilityStatusBadgeClassName(state: IntegrationCapabilityStateState): string {
  switch (state) {
    case "STATE_ENABLED":
      return "border-green-200 bg-green-50 text-green-800 dark:border-green-800 dark:bg-green-950/50 dark:text-green-300";
    case "STATE_DISABLED":
      return "border-red-200 bg-red-50 text-red-800 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300";
    case "STATE_REQUESTED":
      return "border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200";
    case "STATE_AVAILABLE":
      return "border-sky-200 bg-sky-50 text-sky-900 dark:border-sky-800 dark:bg-sky-950/40 dark:text-sky-200";
    case "STATE_UNAVAILABLE":
      return "border-gray-200 bg-gray-100 text-gray-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300";
  }
}

/** Solid circle inside the status badge (matches former standalone dot colors). */
export function getCapabilityStatusBadgeDotClassName(state: IntegrationCapabilityStateState): string {
  switch (state) {
    case "STATE_ENABLED":
      return "bg-green-500 dark:bg-green-400";
    case "STATE_DISABLED":
      return "bg-red-500 dark:bg-red-400";
    case "STATE_REQUESTED":
      return "bg-amber-500 dark:bg-amber-400";
    case "STATE_AVAILABLE":
      return "bg-sky-500 dark:bg-sky-400";
    case "STATE_UNAVAILABLE":
      return "bg-gray-400 dark:bg-gray-500";
  }
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

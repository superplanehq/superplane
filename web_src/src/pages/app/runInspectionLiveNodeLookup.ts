import type { CanvasesCanvasEvent, CanvasesCanvasNodeExecution } from "@/api-client";
import type { SidebarEvent } from "@/ui/componentSidebar/types";

type NodeActivityData = {
  executions: CanvasesCanvasNodeExecution[];
  events: CanvasesCanvasEvent[];
};

function executionTimestamp(execution: CanvasesCanvasNodeExecution): number {
  return Date.parse(execution.updatedAt || execution.createdAt || "");
}

function eventTimestamp(event: CanvasesCanvasEvent): number {
  return Date.parse(event.createdAt || "");
}

function newestByTimestamp<T>(items: T[], timestamp: (item: T) => number): T | null {
  let newest: T | null = null;
  let newestTimestamp = Number.NEGATIVE_INFINITY;

  for (const item of items) {
    const candidateTimestamp = timestamp(item);
    const safeTimestamp = Number.isFinite(candidateTimestamp) ? candidateTimestamp : Number.NEGATIVE_INFINITY;
    if (safeTimestamp > newestTimestamp) {
      newest = item;
      newestTimestamp = safeTimestamp;
    }
  }

  return newest;
}

function runLookupEventFromExecution(nodeId: string, execution: CanvasesCanvasNodeExecution): SidebarEvent | null {
  const executionId = execution.id;
  if (!executionId && !execution.rootEvent?.id) {
    return null;
  }

  return {
    id: executionId || execution.rootEvent!.id!,
    title: "",
    state: "processed",
    isOpen: false,
    nodeId,
    executionId,
    originalExecution: execution,
    kind: "execution",
  };
}

function runLookupEventFromTriggerEvent(nodeId: string, event: CanvasesCanvasEvent): SidebarEvent | null {
  if (!event.id) {
    return null;
  }

  return {
    id: event.id,
    title: "",
    state: "processed",
    isOpen: false,
    nodeId,
    triggerEventId: event.id,
    originalEvent: event,
    kind: "trigger",
  };
}

export function resolveRunLookupEventForNodeActivity(
  nodeId: string,
  nodeType: string,
  nodeData: NodeActivityData,
): SidebarEvent | null {
  if (nodeType === "TYPE_TRIGGER") {
    const latestEvent = newestByTimestamp(nodeData.events, eventTimestamp);
    return latestEvent ? runLookupEventFromTriggerEvent(nodeId, latestEvent) : null;
  }

  const latestExecution = newestByTimestamp(nodeData.executions, executionTimestamp);
  return latestExecution ? runLookupEventFromExecution(nodeId, latestExecution) : null;
}

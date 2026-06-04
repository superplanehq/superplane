import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { flattenObject } from "@/lib/utils";
import { getExecutionDetails, getState, getStateMap } from "@/pages/workflowv2/mappers";
import { buildExecutionInfo } from "@/pages/workflowv2/utils";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";

export type RunNodeDetailTabKey = "details" | "payload" | "configuration";

export type RunNodeDetailTabAvailability = {
  hasDetailsSection: boolean;
  hasPayload: boolean;
  hasConfig: boolean;
};

let lastRunNodeDetailTab: RunNodeDetailTabKey = "details";

export function rememberRunNodeDetailTab(tab: RunNodeDetailTabKey) {
  lastRunNodeDetailTab = tab;
}

export function getLastRunNodeDetailTab(): RunNodeDetailTabKey {
  return lastRunNodeDetailTab;
}

export function isRunNodeDetailTabAvailable(
  tab: RunNodeDetailTabKey,
  availability: RunNodeDetailTabAvailability,
): boolean {
  if (tab === "details") return availability.hasDetailsSection;
  if (tab === "payload") return availability.hasPayload;
  return availability.hasConfig;
}

export function resolveRunNodeDetailTab(
  preferred: RunNodeDetailTabKey,
  availability: RunNodeDetailTabAvailability,
): RunNodeDetailTabKey {
  if (isRunNodeDetailTabAvailable(preferred, availability)) return preferred;
  if (availability.hasDetailsSection) return "details";
  if (availability.hasPayload) return "payload";
  if (availability.hasConfig) return "configuration";
  return preferred;
}

export type RunNodeDetailTabData = {
  details?: Record<string, unknown>;
  payload?: unknown;
  configuration?: unknown;
};

export function workflowComponentName(node: ComponentsNode | undefined): string {
  if (node?.type === "TYPE_ACTION" && node.component) return node.component;
  if (node?.type === "TYPE_TRIGGER" && node.component) return node.component;
  return "default";
}

export function eventBadgeForTriggeredTrigger(node: ComponentsNode | undefined): { badgeColor: string; label: string } {
  const name = workflowComponentName(node);
  const stateMap = getStateMap(name);
  const style = stateMap.triggered ?? DEFAULT_EVENT_STATE_MAP.triggered;
  return { badgeColor: style.badgeColor, label: style.label ?? "triggered" };
}

export function eventBadgeForExecution(
  node: ComponentsNode | undefined,
  execution: CanvasesCanvasNodeExecution,
): { badgeColor: string; label: string } {
  const name = workflowComponentName(node);
  const eventState = getState(name)(buildExecutionInfo(execution));
  const stateMap = getStateMap(name);
  const style = stateMap[eventState] ?? DEFAULT_EVENT_STATE_MAP.neutral;
  return { badgeColor: style.badgeColor, label: style.label ?? String(eventState) };
}

export function buildExecutionChain(executions: CanvasesCanvasNodeExecution[], triggerNodeId?: string | null) {
  const chain: string[] = [];
  const visited = new Set<string>();

  if (triggerNodeId) {
    chain.push(triggerNodeId);
    visited.add(triggerNodeId);
  }

  for (const execution of executions) {
    if (execution.nodeId && !visited.has(execution.nodeId)) {
      visited.add(execution.nodeId);
      chain.push(execution.nodeId);
    }
  }

  return chain;
}

export function getAdjacentRunNodeId(
  chain: string[],
  currentNodeId: string,
  direction: "prev" | "next",
): string | null {
  const currentIndex = chain.indexOf(currentNodeId);
  if (currentIndex === -1) return null;

  const nextIndex = direction === "prev" ? currentIndex - 1 : currentIndex + 1;
  return chain[nextIndex] ?? null;
}

function extractExecutionPayload(execution: CanvasesCanvasNodeExecution): unknown {
  if (!execution.outputs || Object.keys(execution.outputs).length === 0) {
    return undefined;
  }

  const outputData = Object.values(execution.outputs).find((output) => Array.isArray(output) && output.length > 0) as
    | unknown[]
    | undefined;
  if (outputData && outputData.length > 0) {
    return outputData[0];
  }

  return execution.outputs;
}

export function buildExecutionTabData(
  execution: CanvasesCanvasNodeExecution,
  workflowNode: ComponentsNode | undefined,
  workflowNodes: ComponentsNode[],
): RunNodeDetailTabData {
  const tabData: RunNodeDetailTabData = {};
  let details: Record<string, unknown> = {};
  const componentName = typeof workflowNode?.component === "string" ? workflowNode.component : undefined;

  if (componentName && workflowNode) {
    const customDetails = getExecutionDetails(componentName, execution, workflowNode, workflowNodes);
    if (customDetails && Object.keys(customDetails).length > 0) {
      details = { ...customDetails };
    }
  }

  if (Object.keys(details).length === 0) {
    const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
    details = { ...flattenObject((hasOutputs ? execution.outputs : execution.metadata) || {}) };
  }

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED") &&
    !("Error" in details)
  ) {
    details.Error = {
      __type: "error",
      message: execution.resultMessage,
    };
  }

  if (execution.result === "RESULT_CANCELLED" && !("Cancelled by" in details)) {
    const cancelledBy = execution.cancelledBy;
    details["Cancelled by"] = cancelledBy?.name || cancelledBy?.id || "Unknown";
  }

  tabData.details = Object.fromEntries(
    Object.entries(details).filter(([, value]) => value !== undefined && value !== "" && value !== null),
  );

  tabData.payload = extractExecutionPayload(execution);

  if (execution.configuration && Object.keys(execution.configuration).length > 0) {
    tabData.configuration = execution.configuration;
  }

  return tabData;
}

export function buildTriggerTabData(
  run: CanvasesCanvasRun,
  workflowNode: ComponentsNode | undefined,
): RunNodeDetailTabData {
  const details: Record<string, unknown> = {};
  const rootEvent = run.rootEvent;

  if (rootEvent?.channel) details.Channel = rootEvent.channel;
  if (rootEvent?.customName) details.Name = rootEvent.customName;
  if (rootEvent?.createdAt) details["Triggered at"] = rootEvent.createdAt;

  const tabData: RunNodeDetailTabData = {
    details: Object.keys(details).length > 0 ? details : undefined,
    payload: rootEvent?.data && Object.keys(rootEvent.data).length > 0 ? rootEvent.data : undefined,
  };

  if (
    workflowNode?.configuration &&
    typeof workflowNode.configuration === "object" &&
    Object.keys(workflowNode.configuration).length > 0
  ) {
    tabData.configuration = workflowNode.configuration;
  }

  return tabData;
}

export function isErrorValue(value: unknown): value is { __type: "error"; message: string } {
  return !!value && typeof value === "object" && (value as { __type?: string }).__type === "error";
}

export function hasObjectValue(value: unknown) {
  return !!value && typeof value === "object" && Object.keys(value).length > 0;
}

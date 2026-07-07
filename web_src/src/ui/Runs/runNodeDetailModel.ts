import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { flattenObject } from "@/lib/utils";
import { getExecutionDetails, getState, getStateMap, getTriggerRenderer } from "@/pages/app/mappers";
import { buildEventInfo, buildExecutionInfo } from "@/pages/app/utils";
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

export type RunInspectorOutputSection = {
  channel: string;
  value: unknown;
  sizeKb: string;
};

export type RunInspectorUpstreamSection = {
  nodeId: string;
  nodeName: string;
  badge: { badgeColor: string; label: string } | null;
  output: unknown;
};

export type RunInspectorNodeSection = {
  nodeId: string;
  nodeName: string;
  workflowNode?: ComponentsNode;
  execution?: CanvasesCanvasNodeExecution;
  isTrigger: boolean;
  createdAt?: string;
  durationMs?: number;
  badge: { badgeColor: string; label: string } | null;
  tabData: RunNodeDetailTabData | null;
  upstreamSections: RunInspectorUpstreamSection[];
  outputSections: RunInspectorOutputSection[];
  errorMessage?: string;
};

export type RunInspectorErrorSummary = {
  nodeId: string;
  nodeName: string;
  message: string;
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

export function buildRunInspectorNodeSections({
  run,
  executions,
  workflowNodes,
}: {
  run: CanvasesCanvasRun;
  executions: CanvasesCanvasNodeExecution[];
  workflowNodes: ComponentsNode[];
}): RunInspectorNodeSection[] {
  const triggerNodeId = run.rootEvent?.nodeId;
  const executionChain = buildExecutionChain(executions, triggerNodeId);

  return executionChain.map((nodeId) => {
    const workflowNode = workflowNodes.find((node) => node.id === nodeId);
    const execution = executions.find((item) => item.nodeId === nodeId);
    const isTrigger = nodeId === triggerNodeId;
    const tabData = isTrigger
      ? buildTriggerTabData(run, workflowNode)
      : execution
        ? buildExecutionTabData(execution, workflowNode, workflowNodes)
        : null;

    return {
      nodeId,
      nodeName: workflowNode?.name || nodeId,
      workflowNode,
      execution,
      isTrigger,
      createdAt: isTrigger ? run.rootEvent?.createdAt : execution?.createdAt,
      durationMs: execution ? calculateExecutionDuration(execution) : undefined,
      badge: isTrigger
        ? eventBadgeForTriggeredTrigger(workflowNode)
        : execution
          ? eventBadgeForExecution(workflowNode, execution)
          : null,
      tabData,
      upstreamSections: execution ? buildUpstreamSections(execution, executions, workflowNodes) : [],
      outputSections: execution ? buildOutputSections(execution.outputs) : [],
      errorMessage: execution ? getExecutionErrorMessage(execution) : undefined,
    };
  });
}

export function findRunInspectorErrorSummaries(sections: RunInspectorNodeSection[]): RunInspectorErrorSummary[] {
  return sections
    .filter((section) => !!section.errorMessage)
    .map((section) => ({
      nodeId: section.nodeId,
      nodeName: section.nodeName,
      message: section.errorMessage!,
    }));
}

export function calculateRunDuration(run: CanvasesCanvasRun): number | null {
  return calculateDuration(run.createdAt, run.finishedAt || run.updatedAt);
}

function calculateExecutionDuration(execution: CanvasesCanvasNodeExecution): number | undefined {
  return calculateDuration(execution.createdAt, execution.updatedAt) ?? undefined;
}

function calculateDuration(start?: string, end?: string): number | null {
  if (!start || !end) return null;

  const startedAt = new Date(start).getTime();
  const endedAt = new Date(end).getTime();
  if (!Number.isFinite(startedAt) || !Number.isFinite(endedAt) || endedAt < startedAt) return null;

  return endedAt - startedAt;
}

function buildUpstreamSections(
  execution: CanvasesCanvasNodeExecution,
  executions: CanvasesCanvasNodeExecution[],
  workflowNodes: ComponentsNode[],
): RunInspectorUpstreamSection[] {
  if (!execution.previousExecutionId) return [];

  const previousExecution = executions.find((item) => item.id === execution.previousExecutionId);
  if (!previousExecution?.nodeId) return [];

  const workflowNode = workflowNodes.find((node) => node.id === previousExecution.nodeId);

  return [
    {
      nodeId: previousExecution.nodeId,
      nodeName: workflowNode?.name || previousExecution.nodeId,
      badge: eventBadgeForExecution(workflowNode, previousExecution),
      output: previousExecution.outputs,
    },
  ];
}

function buildOutputSections(outputs?: CanvasesCanvasNodeExecution["outputs"]): RunInspectorOutputSection[] {
  if (!outputs || Object.keys(outputs).length === 0) return [];

  return Object.entries(outputs).map(([channel, value]) => ({
    channel,
    value,
    sizeKb: formatJsonSizeKb(value),
  }));
}

function formatJsonSizeKb(value: unknown): string {
  const json = JSON.stringify(value ?? null);
  const bytes = new Blob([json]).size;
  return `${Math.max(bytes / 1024, 0.01).toFixed(2)} KB`;
}

function getExecutionErrorMessage(execution: CanvasesCanvasNodeExecution): string | undefined {
  if (execution.result !== "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR") {
    return undefined;
  }

  return execution.resultMessage || execution.resultReason || "Execution failed";
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

function buildDefaultExecutionDetails(
  execution: CanvasesCanvasNodeExecution,
  workflowNode: ComponentsNode | undefined,
  workflowNodes: ComponentsNode[],
): Record<string, unknown> {
  const componentName = typeof workflowNode?.component === "string" ? workflowNode.component : undefined;

  if (componentName && workflowNode) {
    const customDetails = getExecutionDetails(componentName, execution, workflowNode, workflowNodes);
    if (customDetails && Object.keys(customDetails).length > 0) {
      return { ...customDetails };
    }
  }

  const hasOutputs = execution.outputs && Object.keys(execution.outputs).length > 0;
  return { ...flattenObject((hasOutputs ? execution.outputs : execution.metadata) || {}) };
}

function applyExecutionResultDetails(
  details: Record<string, unknown>,
  execution: CanvasesCanvasNodeExecution,
): Record<string, unknown> {
  const next = { ...details };

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED") &&
    !("Error" in next)
  ) {
    next.Error = {
      __type: "error",
      message: execution.resultMessage,
    };
  }

  if (execution.result === "RESULT_CANCELLED" && !("Cancelled by" in next)) {
    const cancelledBy = execution.cancelledBy;
    next["Cancelled by"] = cancelledBy?.name || cancelledBy?.id || "Unknown";
  }

  return next;
}

function filterEmptyDetailEntries(details: Record<string, unknown>) {
  return Object.fromEntries(
    Object.entries(details).filter(([, value]) => value !== undefined && value !== "" && value !== null),
  );
}

export function buildExecutionTabData(
  execution: CanvasesCanvasNodeExecution,
  workflowNode: ComponentsNode | undefined,
  workflowNodes: ComponentsNode[],
): RunNodeDetailTabData {
  const tabData: RunNodeDetailTabData = {
    details: filterEmptyDetailEntries(
      applyExecutionResultDetails(buildDefaultExecutionDetails(execution, workflowNode, workflowNodes), execution),
    ),
    payload: extractExecutionPayload(execution),
  };

  if (execution.configuration && Object.keys(execution.configuration).length > 0) {
    tabData.configuration = execution.configuration;
  }

  return tabData;
}

export function buildTriggerTabData(
  run: CanvasesCanvasRun,
  workflowNode: ComponentsNode | undefined,
): RunNodeDetailTabData {
  const rootEvent = run.rootEvent;
  const mappedDetails = buildTriggerEventDetails(rootEvent, workflowNode);
  const fallbackDetails = buildFallbackTriggerEventDetails(rootEvent);
  const details = Object.keys(mappedDetails).length > 0 ? mappedDetails : fallbackDetails;

  const tabData: RunNodeDetailTabData = {
    details: Object.keys(details).length > 0 ? filterEmptyDetailEntries(details) : undefined,
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

function buildTriggerEventDetails(
  rootEvent: CanvasesCanvasRun["rootEvent"],
  workflowNode: ComponentsNode | undefined,
): Record<string, unknown> {
  if (!rootEvent) return {};

  const triggerRenderer = getTriggerRenderer(workflowComponentName(workflowNode));
  return triggerRenderer.getRootEventValues({ event: buildEventInfo(rootEvent) });
}

function buildFallbackTriggerEventDetails(rootEvent: CanvasesCanvasRun["rootEvent"]): Record<string, unknown> {
  const details: Record<string, unknown> = {};

  if (rootEvent?.channel) details.Channel = rootEvent.channel;
  if (rootEvent?.customName) details.Name = rootEvent.customName;
  if (rootEvent?.createdAt) details["Triggered at"] = rootEvent.createdAt;

  return details;
}

export function isErrorValue(value: unknown): value is { __type: "error"; message: string } {
  return !!value && typeof value === "object" && (value as { __type?: string }).__type === "error";
}

export function hasObjectValue(value: unknown) {
  return !!value && typeof value === "object" && Object.keys(value).length > 0;
}

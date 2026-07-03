import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { flattenObject } from "@/lib/utils";
import { getExecutionDetails, getState, getStateMap, getTriggerRenderer } from "@/pages/app/mappers";
import { buildEventInfo, buildExecutionInfo } from "@/pages/app/utils";
import { DEFAULT_EVENT_STATE_MAP, type EventState } from "@/ui/componentBase";

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

/** The component-derived event state for an execution (running / queued / success / error / ...). */
export function getExecutionEventState(
  node: ComponentsNode | undefined,
  execution: CanvasesCanvasNodeExecution,
): EventState {
  const name = workflowComponentName(node);
  return getState(name)(buildExecutionInfo(execution));
}

export function eventBadgeForExecution(
  node: ComponentsNode | undefined,
  execution: CanvasesCanvasNodeExecution,
): { badgeColor: string; label: string } {
  const name = workflowComponentName(node);
  const eventState = getExecutionEventState(node, execution);
  const stateMap = getStateMap(name);
  const style = stateMap[eventState] ?? DEFAULT_EVENT_STATE_MAP.neutral;
  return { badgeColor: style.badgeColor, label: style.label ?? String(eventState) };
}

export interface StepStatusEntry {
  key: string;
  label: string;
  /** Tailwind background class for the timeline dot. */
  dotClassName: string;
  /** ISO timestamp for this status change. */
  timestamp: string;
}

/** Event states that mean "blocked, waiting on time or input" (e.g. approval, timegate). */
const WAITING_STATUS_STATES = new Set(["waiting", "queued", "pending"]);

/**
 * The canonical status changes a step goes through, for the Action sub-timeline:
 * Triggered -> Queued -> Running|Waiting -> terminal. The backend only records a
 * single state/result plus created/updated timestamps, so the intermediate
 * timestamps are interpolated between the step's start and finish. The terminal
 * entry's label/color comes from the component's own state map.
 */
export function buildStepStatusTimeline(
  execution: CanvasesCanvasNodeExecution,
  node: ComponentsNode | undefined,
  now: number = Date.now(),
): StepStatusEntry[] {
  const start = execution.createdAt ? new Date(execution.createdAt).getTime() : now;
  const finished = execution.state === "STATE_FINISHED";
  const end = finished && execution.updatedAt ? new Date(execution.updatedAt).getTime() : now;
  const duration = Math.max(0, end - start);
  const at = (fraction: number) => new Date(start + Math.round(duration * fraction)).toISOString();

  const entries: StepStatusEntry[] = [
    { key: "triggered", label: "Triggered", dotClassName: "bg-violet-400", timestamp: new Date(start).toISOString() },
    { key: "queued", label: "Queued", dotClassName: "bg-slate-400", timestamp: at(0.1) },
  ];

  const isWaiting = WAITING_STATUS_STATES.has(getExecutionEventState(node, execution));
  entries.push(
    isWaiting
      ? { key: "waiting", label: "Waiting", dotClassName: "bg-amber-500", timestamp: at(0.25) }
      : { key: "running", label: "Running", dotClassName: "bg-blue-500", timestamp: at(0.25) },
  );

  if (finished) {
    const badge = eventBadgeForExecution(node, execution);
    entries.push({
      key: "terminal",
      label: badge.label,
      dotClassName: badge.badgeColor,
      timestamp: new Date(end).toISOString(),
    });
  }

  return entries;
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

export function extractExecutionPayload(execution: CanvasesCanvasNodeExecution): unknown {
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

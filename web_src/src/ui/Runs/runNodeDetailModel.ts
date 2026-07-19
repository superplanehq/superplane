import type {
  ActionsAction,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasNodeQueueItem,
  CanvasesCanvasRun,
  ComponentsEdge,
  ConfigurationField,
  SuperplaneComponentsNode as ComponentsNode,
  TriggersTrigger,
} from "@/api-client";
import { getState, getStateMap } from "@/pages/app/mappers";
import { buildExecutionInfo } from "@/pages/app/utils";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import {
  buildComponentDefinitionsByName,
  buildTriggerDefinitionsByName,
  resolveConfigurationFields,
} from "./runNodeConfigurationFields";
import {
  buildNodeActions,
  buildOutputSections,
  buildTriggerOutputSections,
  getExecutionErrorMessage,
  normalizeExecutionOutputsForDisplay,
} from "./runNodeDetailOutputs";
import { buildExecutionTabData, buildTriggerTabData } from "./runNodeDetailTabs";
import { buildQueuedNodeSections } from "./runQueuedNodeSections";
export { hasObjectValue } from "./runNodeDetailOutputs";
export { buildExecutionTabData, buildTriggerTabData, isErrorValue } from "./runNodeDetailTabs";

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

export type RunInspectorCurrentUser = {
  id: string;
  email: string;
  roles?: string[];
  groups?: string[];
};

export type RunInspectorApprovalRecord = {
  index: number;
  state?: string;
  type?: string;
  user?: {
    id?: string;
    email?: string;
    name?: string;
  };
  roleRef?: {
    name?: string;
    displayName?: string;
  };
  groupRef?: {
    name?: string;
    displayName?: string;
  };
};

export type RunInspectorNodeActions = {
  canStop: boolean;
  canPushThrough: boolean;
  approvalRecords: RunInspectorApprovalRecord[];
};

export type RunInspectorUpstreamSection = {
  nodeId: string;
  nodeName: string;
  workflowNode?: ComponentsNode;
  badge: { badgeColor: string; label: string } | null;
  output: unknown;
};

export type RunInspectorNodeSection = {
  sectionValue: string;
  nodeId: string;
  nodeName: string;
  workflowNode?: ComponentsNode;
  execution?: CanvasesCanvasNodeExecution;
  executionRef?: CanvasesCanvasNodeExecutionRef;
  queueItem?: CanvasesCanvasNodeQueueItem;
  isTrigger: boolean;
  isQueued: boolean;
  createdAt?: string;
  durationMs?: number;
  badge: { badgeColor: string; label: string } | null;
  tabData: RunNodeDetailTabData | null;
  upstreamSections: RunInspectorUpstreamSection[];
  primaryInputNodeId?: string;
  outputSections: RunInspectorOutputSection[];
  errorMessage?: string;
  actions: RunInspectorNodeActions;
  configurationFields: ConfigurationField[];
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

export function eventBadgeForExecutionRef(
  node: ComponentsNode | undefined,
  executionRef: CanvasesCanvasNodeExecutionRef,
): { badgeColor: string; label: string } {
  return eventBadgeForExecution(node, {
    id: executionRef.id,
    nodeId: executionRef.nodeId,
    state: executionRef.state ?? "STATE_UNKNOWN",
    result: executionRef.result ?? "RESULT_UNKNOWN",
    resultReason: executionRef.resultReason ?? "RESULT_REASON_OK",
    resultMessage: executionRef.resultMessage ?? "",
    createdAt: executionRef.createdAt,
    updatedAt: executionRef.updatedAt,
    outputs: {},
    metadata: {},
    configuration: {},
  });
}

export function buildExecutionChain(
  executions: CanvasesCanvasNodeExecution[],
  triggerNodeId?: string | null,
  executionRefs: CanvasesCanvasNodeExecutionRef[] = [],
) {
  const chain: string[] = [];
  const visited = new Set<string>();

  if (triggerNodeId) {
    chain.push(triggerNodeId);
    visited.add(triggerNodeId);
  }

  for (const executionRef of [...executionRefs, ...executions]) {
    if (executionRef.nodeId && !visited.has(executionRef.nodeId)) {
      visited.add(executionRef.nodeId);
      chain.push(executionRef.nodeId);
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
  workflowEdges,
  componentDefinitions,
  triggerDefinitions,
}: {
  run: CanvasesCanvasRun;
  executions: CanvasesCanvasNodeExecution[];
  workflowNodes: ComponentsNode[];
  workflowEdges?: ComponentsEdge[];
  componentDefinitions?: ActionsAction[];
  triggerDefinitions?: TriggersTrigger[];
}): RunInspectorNodeSection[] {
  const triggerNodeId = run.rootEvent?.nodeId;
  const executionChain = buildExecutionChain(executions, triggerNodeId, run.executions);
  const executionIndexByNodeId = new Map(executionChain.map((nodeId, index) => [nodeId, index]));
  const componentDefinitionsByName = buildComponentDefinitionsByName(componentDefinitions);
  const triggerDefinitionsByName = buildTriggerDefinitionsByName(triggerDefinitions);

  const executionSections = executionChain.map((nodeId, index) => {
    const workflowNode = workflowNodes.find((node) => node.id === nodeId);
    const execution = executions.find((item) => item.nodeId === nodeId);
    const executionRef = run.executions?.find((item) => item.nodeId === nodeId) ?? execution;
    const errorSource = execution ?? executionRef;
    const isTrigger = nodeId === triggerNodeId;
    const tabData = isTrigger
      ? buildTriggerTabData(run, workflowNode)
      : execution
        ? buildExecutionTabData(execution, workflowNode, workflowNodes, executionRef)
        : null;
    const upstreamSections = isTrigger
      ? []
      : buildUpstreamSections({
          executionChain,
          currentIndex: index,
          run,
          executions,
          executionRefs: run.executions ?? [],
          workflowNodes,
          workflowEdges,
          executionIndexByNodeId,
        });

    return {
      sectionValue: nodeId,
      nodeId,
      nodeName: workflowNode?.name || nodeId,
      workflowNode,
      execution,
      executionRef,
      isTrigger,
      isQueued: false,
      createdAt: isTrigger ? run.rootEvent?.createdAt : (execution?.createdAt ?? executionRef?.createdAt),
      durationMs: execution
        ? calculateExecutionDuration(execution)
        : executionRef
          ? calculateExecutionRefDuration(executionRef)
          : undefined,
      badge: isTrigger
        ? eventBadgeForTriggeredTrigger(workflowNode)
        : execution
          ? eventBadgeForExecution(workflowNode, execution)
          : executionRef
            ? eventBadgeForExecutionRef(workflowNode, executionRef)
            : null,
      tabData,
      upstreamSections,
      primaryInputNodeId: isTrigger
        ? undefined
        : findPrimaryInputNodeId({
            executionChain,
            currentIndex: index,
            run,
            executions,
            executionRefs: run.executions ?? [],
            workflowEdges,
            executionIndexByNodeId,
          }),
      outputSections: isTrigger
        ? buildTriggerOutputSections(run)
        : execution
          ? buildOutputSections(execution.outputs)
          : [],
      errorMessage: errorSource ? getExecutionErrorMessage(errorSource) : undefined,
      actions: buildNodeActions(workflowNode, execution),
      configurationFields: resolveConfigurationFields({
        workflowNode,
        componentDefinitionsByName,
        triggerDefinitionsByName,
      }),
    };
  });

  return [...executionSections, ...buildQueuedNodeSections({ run, workflowNodes })];
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

function calculateExecutionRefDuration(execution: CanvasesCanvasNodeExecutionRef): number | undefined {
  return calculateDuration(execution.createdAt, execution.updatedAt) ?? undefined;
}

function calculateDuration(start?: string, end?: string): number | null {
  if (!start || !end) return null;

  const startedAt = new Date(start).getTime();
  const endedAt = new Date(end).getTime();
  if (!Number.isFinite(startedAt) || !Number.isFinite(endedAt) || endedAt < startedAt) return null;

  return endedAt - startedAt;
}

function buildUpstreamSections({
  executionChain,
  currentIndex,
  run,
  executions,
  executionRefs,
  workflowNodes,
  workflowEdges,
  executionIndexByNodeId,
}: {
  executionChain: string[];
  currentIndex: number;
  run: CanvasesCanvasRun;
  executions: CanvasesCanvasNodeExecution[];
  executionRefs: CanvasesCanvasNodeExecutionRef[];
  workflowNodes: ComponentsNode[];
  workflowEdges?: ComponentsEdge[];
  executionIndexByNodeId: Map<string, number>;
}): RunInspectorUpstreamSection[] {
  if (currentIndex <= 0) return [];

  let orderedNodeIds: string[];
  if (!hasWorkflowEdges(workflowEdges)) {
    orderedNodeIds = executionChain.slice(0, currentIndex);
  } else {
    orderedNodeIds = findAccessibleUpstreamNodeIds(
      executionChain[currentIndex],
      workflowEdges,
      executionIndexByNodeId,
    ).sort((left, right) => compareUpstreamCreatedAt(left, right, run, executions, executionRefs));
  }

  return orderedNodeIds.map((nodeId) => {
    const workflowNode = workflowNodes.find((node) => node.id === nodeId);
    const execution = executions.find((item) => item.nodeId === nodeId);
    const isTrigger = nodeId === run.rootEvent?.nodeId;

    return {
      nodeId,
      nodeName: workflowNode?.name || nodeId,
      workflowNode,
      badge: isTrigger
        ? eventBadgeForTriggeredTrigger(workflowNode)
        : execution
          ? eventBadgeForExecution(workflowNode, execution)
          : null,
      output: isTrigger ? run.rootEvent?.data : normalizeExecutionOutputsForDisplay(execution?.outputs),
    };
  });
}

function findPrimaryInputNodeId({
  executionChain,
  currentIndex,
  run,
  executions,
  executionRefs,
  workflowEdges,
  executionIndexByNodeId,
}: {
  executionChain: string[];
  currentIndex: number;
  run: CanvasesCanvasRun;
  executions: CanvasesCanvasNodeExecution[];
  executionRefs: CanvasesCanvasNodeExecutionRef[];
  workflowEdges?: ComponentsEdge[];
  executionIndexByNodeId: Map<string, number>;
}): string | undefined {
  if (currentIndex <= 0) return undefined;

  if (!hasWorkflowEdges(workflowEdges)) {
    return executionChain[currentIndex - 1];
  }

  const currentNodeId = executionChain[currentIndex];
  const directInputNodeIds = workflowEdges
    .filter((edge) => edge.targetId === currentNodeId && edge.sourceId)
    .map((edge) => edge.sourceId!)
    .filter((nodeId) => {
      const executionIndex = executionIndexByNodeId.get(nodeId);
      return executionIndex !== undefined && executionIndex < currentIndex;
    });

  if (directInputNodeIds.length > 0) {
    return directInputNodeIds
      .sort((left, right) => compareUpstreamCreatedAt(left, right, run, executions, executionRefs))
      .at(-1);
  }

  return findAccessibleUpstreamNodeIds(currentNodeId, workflowEdges, executionIndexByNodeId)
    .sort((left, right) => compareUpstreamCreatedAt(left, right, run, executions, executionRefs))
    .at(-1);
}

function hasWorkflowEdges(workflowEdges: ComponentsEdge[] | undefined): workflowEdges is ComponentsEdge[] {
  return Boolean(workflowEdges?.length);
}

function findAccessibleUpstreamNodeIds(
  nodeId: string,
  workflowEdges: ComponentsEdge[],
  executionIndexByNodeId: Map<string, number>,
): string[] {
  const accessibleNodeIds = new Set<string>();
  const visited = new Set<string>();
  const pending = [nodeId];
  const currentNodeIndex = executionIndexByNodeId.get(nodeId) ?? Number.POSITIVE_INFINITY;

  while (pending.length > 0) {
    const currentNodeId = pending.pop();
    if (!currentNodeId || visited.has(currentNodeId)) continue;

    visited.add(currentNodeId);

    workflowEdges
      .filter((edge) => edge.targetId === currentNodeId && edge.sourceId)
      .forEach((edge) => {
        const sourceId = edge.sourceId!;
        const sourceIndex = executionIndexByNodeId.get(sourceId);
        if (sourceIndex !== undefined && sourceIndex < currentNodeIndex) {
          accessibleNodeIds.add(sourceId);
        }
        if (!visited.has(sourceId)) {
          pending.push(sourceId);
        }
      });
  }

  return Array.from(accessibleNodeIds);
}

function compareUpstreamCreatedAt(
  leftNodeId: string,
  rightNodeId: string,
  run: CanvasesCanvasRun,
  executions: CanvasesCanvasNodeExecution[],
  executionRefs: CanvasesCanvasNodeExecutionRef[],
): number {
  return (
    getUpstreamCreatedAt(leftNodeId, run, executions, executionRefs) -
    getUpstreamCreatedAt(rightNodeId, run, executions, executionRefs)
  );
}

function getUpstreamCreatedAt(
  nodeId: string,
  run: CanvasesCanvasRun,
  executions: CanvasesCanvasNodeExecution[],
  executionRefs: CanvasesCanvasNodeExecutionRef[],
): number {
  const createdAt =
    nodeId === run.rootEvent?.nodeId
      ? run.rootEvent?.createdAt
      : (executions.find((item) => item.nodeId === nodeId)?.createdAt ??
        executionRefs.find((item) => item.nodeId === nodeId)?.createdAt);

  const timestamp = createdAt ? new Date(createdAt).getTime() : Number.POSITIVE_INFINITY;
  return Number.isFinite(timestamp) ? timestamp : Number.POSITIVE_INFINITY;
}

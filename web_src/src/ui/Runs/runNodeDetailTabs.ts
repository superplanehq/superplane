import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
  SuperplaneComponentsNode,
} from "@/api-client";
import { flattenObject } from "@/lib/utils";
import { getExecutionDetails, getTriggerRenderer } from "@/pages/app/mappers";
import { buildEventInfo } from "@/pages/app/utils";
import { hasObjectValue, normalizeExecutionOutputsForDisplay } from "./runNodeDetailOutputs";
import { workflowComponentName } from "./runNodeDetailModel";
import type { RunNodeDetailTabData } from "./types";

export function buildExecutionTabData(
  execution: CanvasesCanvasNodeExecution,
  workflowNode: SuperplaneComponentsNode | undefined,
  workflowNodes: SuperplaneComponentsNode[],
  executionRef?: CanvasesCanvasNodeExecutionRef,
): RunNodeDetailTabData {
  const tabData: RunNodeDetailTabData = {
    details: filterEmptyDetailEntries(
      applyExecutionResultDetails(
        buildDefaultExecutionDetails(execution, workflowNode, workflowNodes, executionRef),
        execution,
      ),
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
  workflowNode: SuperplaneComponentsNode | undefined,
): RunNodeDetailTabData {
  const rootEvent = run.rootEvent;
  const mappedDetails = buildTriggerEventDetails(rootEvent, workflowNode);
  const fallbackDetails = buildFallbackTriggerEventDetails(rootEvent);
  const details = Object.keys(mappedDetails).length > 0 ? mappedDetails : fallbackDetails;

  const tabData: RunNodeDetailTabData = {
    details: Object.keys(details).length > 0 ? filterEmptyDetailEntries(details) : undefined,
    payload: hasObjectValue(rootEvent?.data) ? rootEvent.data : undefined,
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
  workflowNode: SuperplaneComponentsNode | undefined,
  workflowNodes: SuperplaneComponentsNode[],
  executionRef?: CanvasesCanvasNodeExecutionRef,
): Record<string, unknown> {
  const componentName = typeof workflowNode?.component === "string" ? workflowNode.component : undefined;

  if (componentName && workflowNode) {
    const customDetails = getExecutionDetails(
      componentName,
      execution,
      workflowNode,
      workflowNodes,
      executionRef?.runs,
    );
    if (customDetails && Object.keys(customDetails).length > 0) {
      return { ...customDetails };
    }
  }

  const displayOutputs = normalizeExecutionOutputsForDisplay(execution.outputs);
  return { ...flattenObject((displayOutputs ?? execution.metadata) || {}) };
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

function buildTriggerEventDetails(
  rootEvent: CanvasesCanvasRun["rootEvent"],
  workflowNode: SuperplaneComponentsNode | undefined,
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

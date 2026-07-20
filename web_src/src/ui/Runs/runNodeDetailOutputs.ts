import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import type {
  RunInspectorApprovalRecord,
  RunInspectorNodeActions,
  RunInspectorOutputSection,
} from "./types";

export function buildTriggerOutputSections(run: CanvasesCanvasRun): RunInspectorOutputSection[] {
  if (!hasObjectValue(run.rootEvent?.data)) return [];

  return [
    {
      channel: run.rootEvent?.channel || "default",
      value: run.rootEvent?.data,
      sizeKb: formatJsonSizeKb(run.rootEvent?.data),
    },
  ];
}

export function buildOutputSections(outputs?: CanvasesCanvasNodeExecution["outputs"]): RunInspectorOutputSection[] {
  if (!outputs || Object.keys(outputs).length === 0) return [];

  return Object.entries(outputs).map(([channel, value]) => {
    const displayValue = normalizeExecutionChannelOutput(value);

    return {
      channel,
      value: displayValue,
      sizeKb: formatJsonSizeKb(displayValue),
    };
  });
}

export function normalizeExecutionOutputsForDisplay(outputs?: CanvasesCanvasNodeExecution["outputs"]): unknown {
  const outputSections = buildOutputSections(outputs);
  if (outputSections.length === 0) return undefined;
  if (outputSections.length === 1) return outputSections[0].value;

  return Object.fromEntries(outputSections.map((section) => [section.channel, section.value]));
}

export function buildNodeActions(
  workflowNode: ComponentsNode | undefined,
  execution: CanvasesCanvasNodeExecution | undefined,
): RunInspectorNodeActions {
  const isActive = execution?.state === "STATE_STARTED" || execution?.state === "STATE_PENDING";
  const componentName = normalizeComponentName(workflowNode?.component);

  return {
    canStop: Boolean(execution?.id && isActive),
    canPushThrough: Boolean(execution?.id && isActive && (componentName === "wait" || componentName === "timegate")),
    approvalRecords:
      execution?.id && isActive && componentName.includes("approval") ? extractApprovalRecords(execution.metadata) : [],
  };
}

export function hasObjectValue(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && Object.keys(value).length > 0;
}

function normalizeExecutionChannelOutput(value: unknown): unknown {
  if (!Array.isArray(value)) return value;
  return value[0];
}

function formatJsonSizeKb(value: unknown): string {
  const json = JSON.stringify(value ?? null);
  const bytes = new Blob([json]).size;
  return `${Math.max(bytes / 1024, 0.01).toFixed(2)} KB`;
}

function normalizeComponentName(componentName: string | undefined): string {
  return (componentName || "").replace(/[^a-zA-Z0-9]/g, "").toLowerCase();
}

function extractApprovalRecords(metadata: CanvasesCanvasNodeExecution["metadata"]): RunInspectorApprovalRecord[] {
  const records = metadata?.records;
  if (!Array.isArray(records)) return [];

  return records.filter(isApprovalRecord).map((record) => ({
    index: record.index,
    state: typeof record.state === "string" ? record.state : undefined,
    type: typeof record.type === "string" ? record.type : undefined,
    user: isObjectRecord(record.user)
      ? {
          id: typeof record.user.id === "string" ? record.user.id : undefined,
          email: typeof record.user.email === "string" ? record.user.email : undefined,
          name: typeof record.user.name === "string" ? record.user.name : undefined,
        }
      : undefined,
    roleRef: parseApprovalRecordRef(record.roleRef),
    groupRef: parseApprovalRecordRef(record.groupRef),
  }));
}

function parseApprovalRecordRef(value: unknown): { name?: string; displayName?: string } | undefined {
  if (!isObjectRecord(value)) return undefined;

  return {
    name: typeof value.name === "string" ? value.name : undefined,
    displayName: typeof value.displayName === "string" ? value.displayName : undefined,
  };
}

function isApprovalRecord(value: unknown): value is {
  index: number;
  state?: unknown;
  type?: unknown;
  user?: unknown;
  roleRef?: unknown;
  groupRef?: unknown;
} {
  return isObjectRecord(value) && typeof value.index === "number";
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object";
}

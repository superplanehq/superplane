import type React from "react";
import type {
  ComponentBaseContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "../../types";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { stringOrDash } from "../../utils";
import prometheusIcon from "@/assets/icons/integrations/aws.prometheus.svg";

export const MAX_METADATA_ITEMS = 3;

export interface WorkspaceStatus {
  statusCode?: string;
}

export interface PrometheusWorkspace {
  alias?: string;
  arn?: string;
  kmsKeyArn?: string;
  prometheusEndpoint?: string;
  status?: WorkspaceStatus;
  tags?: Record<string, string>;
  workspaceId?: string;
}

export interface WorkspaceOutput {
  workspace?: PrometheusWorkspace;
}

export interface WorkspaceNodeMetadata {
  region?: string;
  workspaceId?: string;
  workspaceAlias?: string;
}

export interface QueryConfiguration {
  region?: string;
  workspace?: string;
  query?: string;
}

export interface QueryRangeConfiguration extends QueryConfiguration {
  start?: string;
  end?: string;
  step?: string;
}

export interface PrometheusQueryPayload {
  resultType?: string;
  result?: unknown;
}

export function buildPrometheusComponentProps(
  context: ComponentBaseContext,
  metadata: MetadataItem[],
): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name || "unknown";

  return {
    title:
      context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component",
    iconSrc: prometheusIcon,
    iconColor: getColorClass(context.componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
    collapsed: context.node.isCollapsed,
    eventSections: lastExecution
      ? buildPrometheusEventSections(context.nodes, lastExecution, componentName)
      : undefined,
    includeEmptyState: !lastExecution,
    metadata,
    eventStateMap: getStateMap(componentName),
  };
}

export function buildPrometheusEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt ?? 0)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent?.id ?? "",
    },
  ];
}

export function prometheusSubtitle(context: SubtitleContext): string | React.ReactNode {
  if (!context.execution.createdAt) {
    return "";
  }

  return renderTimeAgo(new Date(context.execution.createdAt));
}

export function firstOutputData<T>(outputs: unknown): T | undefined {
  const typedOutputs = outputs as { default?: OutputPayload[] } | undefined;
  return typedOutputs?.default?.[0]?.data as T | undefined;
}

export function workspaceExecutionDetails(
  workspace: PrometheusWorkspace | undefined,
  execution: ExecutionInfo,
  timestampLabel: string,
  fallbackAlias?: string,
  timestampSource: "created" | "completed" = "completed",
): Record<string, string> {
  const details: Record<string, string> = {
    [timestampLabel]: stringOrDash(formatExecutionTimestamp(execution, timestampSource)),
  };

  if (!workspace) {
    details.Alias = stringOrDash(fallbackAlias);
    return details;
  }

  details.Alias = stringOrDash(workspace.alias ?? fallbackAlias);
  details.Status = stringOrDash(workspace.status?.statusCode);
  details.ARN = stringOrDash(workspace.arn);

  if (workspace.prometheusEndpoint) {
    details["Prometheus Endpoint"] = workspace.prometheusEndpoint;
  }
  if (workspace.kmsKeyArn) {
    details["KMS Key ARN"] = workspace.kmsKeyArn;
  }

  return details;
}

export function formatExecutionTimestamp(
  execution: ExecutionInfo,
  timestampSource: "created" | "completed" = "completed",
): string | undefined {
  const timestamp = timestampSource === "created" ? execution.createdAt : execution.updatedAt || execution.createdAt;
  if (!timestamp) {
    return undefined;
  }

  return new Date(timestamp).toLocaleString();
}

export function workspaceAliasFromMetadata(node: NodeInfo): string | undefined {
  const metadata = node.metadata as WorkspaceNodeMetadata | undefined;
  return metadata?.workspaceAlias?.trim() || undefined;
}

export function queryMetadataList(node: NodeInfo, range = false): MetadataItem[] {
  const config = (node.configuration ?? {}) as QueryRangeConfiguration;
  const items = [
    metadataItem("activity", workspaceMetadataLabel(node, config)),
    metadataItem("search", config.query),
    rangeMetadataItem(config, range),
  ];

  return items.filter(isMetadataItem).slice(0, MAX_METADATA_ITEMS);
}

function workspaceMetadataLabel(node: NodeInfo, config: QueryConfiguration): string | undefined {
  return workspaceAliasFromMetadata(node) ?? config.workspace;
}

function rangeMetadataItem(config: QueryRangeConfiguration, range: boolean): MetadataItem | undefined {
  if (range) {
    return metadataItem("clock", startMetadataLabel(config.start));
  }

  return metadataItem("globe", config.region);
}

function startMetadataLabel(start: string | undefined): string | undefined {
  if (!start) {
    return undefined;
  }

  return `Start: ${start}`;
}

function metadataItem(icon: MetadataItem["icon"], label: string | undefined): MetadataItem | undefined {
  if (!label) {
    return undefined;
  }

  return { icon, label };
}

function isMetadataItem(item: MetadataItem | undefined): item is MetadataItem {
  return item !== undefined;
}

export function queryDetails(
  execution: ExecutionInfo,
  node: NodeInfo,
  payload: PrometheusQueryPayload | undefined,
): Record<string, string> {
  const config = node.configuration as QueryRangeConfiguration | undefined;
  const details: Record<string, string> = {
    "Executed At": stringOrDash(formatExecutionTimestamp(execution)),
    Alias: stringOrDash(workspaceAliasFromMetadata(node) ?? config?.workspace),
  };

  details["Result Type"] = stringOrDash(payload?.resultType);
  details.Results = resultCount(payload);

  return details;
}

function resultCount(payload: PrometheusQueryPayload | undefined): string {
  if (payload?.result === undefined) {
    return "-";
  }

  if (isSingleValueResult(payload.resultType)) {
    return "1";
  }

  return String(Array.isArray(payload.result) ? payload.result.length : 0);
}

function isSingleValueResult(resultType: string | undefined): boolean {
  return resultType === "scalar" || resultType === "string";
}

export function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Workspace",
    componentName: "prometheus.getWorkspace",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

export function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-06-08T09:00:00Z").toISOString(),
    updatedAt: new Date("2026-06-08T09:01:00Z").toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

export function buildOutput(data: Record<string, unknown>) {
  return { type: "json", timestamp: new Date().toISOString(), data };
}

export function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

export function buildComponentCtx(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildNode(nodeOverrides);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: node.componentName,
      label: "Prometheus • Workspace",
      description: "",
      icon: "aws",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

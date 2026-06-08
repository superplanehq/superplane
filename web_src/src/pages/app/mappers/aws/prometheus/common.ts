import type React from "react";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo, OutputPayload, SubtitleContext } from "../../types";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import type { MetadataItem } from "@/ui/metadataList";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import { stringOrDash } from "../../utils";
import prometheusIcon from "@/assets/icons/integrations/prometheus.svg";

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

export function workspaceExecutionDetails(workspace?: PrometheusWorkspace): Record<string, string> {
  if (!workspace) {
    return {};
  }

  const details: Record<string, string> = {
    "Workspace ID": stringOrDash(workspace.workspaceId),
    Alias: stringOrDash(workspace.alias),
    Status: stringOrDash(workspace.status?.statusCode),
    ARN: stringOrDash(workspace.arn),
  };

  if (workspace.prometheusEndpoint) {
    details["Prometheus Endpoint"] = workspace.prometheusEndpoint;
  }
  if (workspace.kmsKeyArn) {
    details["KMS Key ARN"] = workspace.kmsKeyArn;
  }

  return details;
}

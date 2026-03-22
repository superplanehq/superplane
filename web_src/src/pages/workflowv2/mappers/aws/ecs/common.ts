import type { ComponentBaseContext, ExecutionInfo, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsEcsIcon from "@/assets/icons/integrations/aws.ecs.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

export const MAX_METADATA_ITEMS = 3;

/** ECS Console URL for a cluster (and optional service or task). */
export function ecsConsoleUrl(region: string, cluster: string, service?: string, taskArn?: string): string {
  const base = `https://${region}.console.aws.amazon.com/ecs/v2/clusters/${encodeURIComponent(cluster)}`;
  if (taskArn) {
    const taskId = taskArn.split("/").pop() ?? "";
    return `${base}/tasks/${encodeURIComponent(taskId)}`;
  }
  if (service) {
    return `${base}/services/${encodeURIComponent(service)}`;
  }
  return base;
}

export function buildEcsComponentProps(
  context: ComponentBaseContext,
  metadata: MetadataItem[],
  eventSections?: EventSection[],
): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name || "unknown";

  return {
    title:
      context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component",
    iconSrc: awsEcsIcon,
    iconColor: getColorClass(context.componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
    collapsed: context.node.isCollapsed,
    eventSections: lastExecution
      ? eventSections || buildEcsEventSections(context.nodes, lastExecution, componentName)
      : undefined,
    includeEmptyState: !lastExecution,
    metadata,
    eventStateMap: getStateMap(componentName),
  };
}

export function buildEcsEventSections(
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

export function ecsSubtitle(context: SubtitleContext): string | React.ReactNode {
  if (!context.execution.createdAt) {
    return "";
  }
  return renderTimeAgo(new Date(context.execution.createdAt));
}

export function truncateForDisplay(value: string, maxLen = 40): string {
  if (!value || value.length <= maxLen) return value;
  return value.substring(0, maxLen) + "...";
}

import type { ComponentBaseContext, ExecutionInfo, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { ComponentBaseProps, EventSection } from "@/pages/workflowv2/mappers/rendererTypes";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { getState, getStateMap } from "../..";
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
  _nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  return [
    {
      receivedAt: new Date(execution.createdAt ?? 0),
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

export function truncateForDisplay(value: unknown, maxLen = 40): string {
  const str = typeof value === "string" ? value : value == null ? "" : String(value);
  if (!str || str.length <= maxLen) return str;
  return str.substring(0, maxLen) + "...";
}

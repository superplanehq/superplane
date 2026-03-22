import type { ComponentBaseContext, ExecutionInfo, NodeInfo, SubtitleContext } from "../../types";
import type React from "react";
import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { getState, getStateMap, getTriggerRenderer } from "../..";
import awsSnsIcon from "@/assets/icons/integrations/aws.sns.svg";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { MetadataItem } from "@/ui/metadataList";

export function buildSnsProps(context: ComponentBaseContext, metadata: MetadataItem[]): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name || "unknown";

  return {
    title: context.node.name || context.componentDefinition.label || "Unnamed component",
    iconSrc: awsSnsIcon,
    iconColor: getColorClass(context.componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
    collapsed: context.node.isCollapsed,
    eventSections: lastExecution ? buildEventSections(context.nodes, lastExecution, componentName) : undefined,
    includeEmptyState: !lastExecution,
    metadata,
    eventStateMap: getStateMap(componentName),
  };
}

export function buildSubtitle(context: SubtitleContext): string | React.ReactNode {
  if (!context.execution.createdAt) {
    return "";
  }

  return renderTimeAgo(new Date(context.execution.createdAt));
}

export function buildEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  if (!execution.createdAt || !execution.rootEvent?.id) {
    return [];
  }

  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent.id,
    },
  ];
}

export function extractArnResourceName(arn?: string): string | undefined {
  if (!arn) {
    return undefined;
  }

  const name = arn.split(":").at(-1);
  return name || undefined;
}

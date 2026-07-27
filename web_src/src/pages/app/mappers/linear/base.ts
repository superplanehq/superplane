import type { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import { getState, getStateMap, getTriggerRenderer } from "..";
import linearIcon from "@/assets/icons/integrations/linear.svg";
import type { ComponentBaseContext, ExecutionInfo, NodeInfo } from "../types";
import type { MetadataItem } from "@/ui/metadataList";

export function linearComponentBaseProps(context: ComponentBaseContext, metadata: MetadataItem[]): ComponentBaseProps {
  const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
  const componentName = context.componentDefinition.name || "linear";

  return {
    iconSrc: linearIcon,
    collapsedBackground: getBackgroundColorClass(context.componentDefinition.color),
    collapsed: context.node.isCollapsed,
    title: context.node.name || context.componentDefinition.label || "Unnamed component",
    eventSections: lastExecution ? linearEventSections(context.nodes, lastExecution, componentName) : undefined,
    metadata,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function linearEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) return [];

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) return [];

  const { title } = getTriggerRenderer(rootTriggerNode.componentName).getTitleAndSubtitle({ event: rootEvent });

  return [
    {
      receivedAt: new Date(execution.createdAt),
      eventTitle: title,
      eventSubtitle: renderTimeAgo(new Date(execution.createdAt)),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}

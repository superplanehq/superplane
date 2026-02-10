import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import renderIcon from "@/assets/icons/integrations/render.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentDefinition, ExecutionInfo, NodeInfo } from "../types";

export function baseProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: ComponentDefinition,
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "render.unknown";

  return {
    title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
    iconSrc: renderIcon,
    iconColor: getColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootTriggerNode = nodes.find((node) => node.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";

  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId: execution.rootEvent!.id!,
    },
  ];
}

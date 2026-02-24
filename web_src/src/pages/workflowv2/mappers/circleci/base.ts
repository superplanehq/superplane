import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getBackgroundColorClass, getColorClass } from "@/utils/colors";
import { formatTimeAgo } from "@/utils/date";
import circleCIIcon from "@/assets/icons/integrations/circleci.svg";
import { getState, getStateMap, getTriggerRenderer } from "..";
import { ComponentDefinition, ExecutionInfo, NodeInfo } from "../types";

export function baseProps(
  nodes: NodeInfo[],
  node: NodeInfo,
  componentDefinition: ComponentDefinition,
  lastExecutions: ExecutionInfo[],
): ComponentBaseProps {
  const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
  const componentName = componentDefinition.name || node.componentName || "circleci.unknown";

  return {
    title: node.name || componentDefinition.label || componentDefinition.name || "Unnamed component",
    iconSrc: circleCIIcon,
    iconColor: getColorClass(componentDefinition.color),
    collapsedBackground: getBackgroundColorClass(componentDefinition.color),
    collapsed: node.isCollapsed,
    eventSections: lastExecution ? baseEventSections(nodes, lastExecution, componentName) : undefined,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function baseEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  const rootTriggerNode = nodes.find((node) => node.id === rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName || "");
  const eventTitle = rootEvent ? rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent }).title : "Event";

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? formatTimeAgo(new Date(subtitleTimestamp)) : "";
  const eventId = rootEvent?.id || execution.id;

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      eventTitle,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId,
    },
  ];
}

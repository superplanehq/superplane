import type { ComponentBaseProps, EventSection } from "@/pages/workflowv2/mappers/types";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import circleCIIcon from "@/assets/icons/integrations/circleci.svg";
import { getState, getStateMap } from "..";
import type { ComponentDefinition, ExecutionInfo, NodeInfo } from "../types";

export function baseProps(
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
    eventSections: lastExecution ? baseEventSections(lastExecution, componentName) : undefined,
    includeEmptyState: !lastExecution,
    eventStateMap: getStateMap(componentName),
  };
}

function baseEventSections(execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;

  const subtitleTimestamp = execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : "";
  const eventId = rootEvent?.id || execution.id;

  return [
    {
      receivedAt: execution.createdAt ? new Date(execution.createdAt) : undefined,
      eventSubtitle,
      eventState: getState(componentName)(execution),
      eventId,
    },
  ];
}

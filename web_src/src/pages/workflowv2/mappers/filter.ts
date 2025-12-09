import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection, EventState } from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { getBackgroundColorClass } from "@/utils/colors";
import { parseExpression } from "@/lib/expressionParser";

export const filterMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _queueItems: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "filter";
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const expression = (node.configuration?.expression as string) || "";
    const filters = parseExpression(expression);
    const specs = expression
      ? [
          {
            title: "filter",
            tooltipTitle: "filters applied",
            values: filters,
          },
        ]
      : undefined;

    return {
      iconSlug: "filter",
      headerColor: "bg-gray-50",
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getfilterEventSections(nodes, lastExecution, componentName),
      specs,
      eventStateMap: getStateMap(componentName),
    };
  },
};

function getfilterEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution | null,
  componentName: string,
): EventSection[] {
  let lastEvent: Omit<EventSection, "title"> = {
    eventTitle: "No events received yet",
    eventState: "neutral" as const,
  };
  if (execution) {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    lastEvent = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
    };
  }

  const eventSections: EventSection[] = [];
  if (lastEvent) {
    eventSections.push({
      title: "Last Event",
      ...lastEvent,
    });
  }

  return eventSections;
}

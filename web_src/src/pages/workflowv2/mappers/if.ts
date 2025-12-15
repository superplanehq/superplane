import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection } from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { parseExpression } from "@/lib/expressionParser";

export const ifMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _nodeQueueItems: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "if";
    const expression = node.configuration?.expression as string | undefined;
    const conditions = expression ? parseExpression(expression) : [];
    const specs = expression
      ? [
          {
            title: "condition",
            tooltipTitle: "conditions applied",
            values: conditions,
          },
        ]
      : undefined;

    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;

    return {
      iconSlug: "split",
      headerColor: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: lastExecution ? getEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs: specs,
      runDisabled: false,
      runDisabledTooltip: undefined,
      eventStateMap: getStateMap(componentName),
    };
  },
};

function getEventSections(
  nodes: ComponentsNode[],
  execution: WorkflowsWorkflowNodeExecution,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

  const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventState: getState(componentName)(execution),
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}

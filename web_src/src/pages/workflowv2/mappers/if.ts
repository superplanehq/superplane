import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection, EventState } from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { parseExpression } from "@/lib/expressionParser";

type IfEvent = {
  eventTitle: string;
  eventState: EventState;
  receivedAt?: Date;
};

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

    return {
      iconSlug: "split",
      headerColor: "bg-white",
      collapsed: node.isCollapsed,
      collapsedBackground: "bg-white",
      title: node.name!,
      eventSections: getEventSections(nodes, lastExecutions, componentName),
      specs: specs,
      runDisabled: false,
      runDisabledTooltip: undefined,
      eventStateMap: getStateMap(componentName),
    };
  },
};

function getEventSections(
  nodes: ComponentsNode[],
  executions: WorkflowsWorkflowNodeExecution[],
  componentName: string,
): EventSection[] {
  const lastTrueExecution = executions.length > 0 ? executions.find((e) => e.outputs?.["true"]) : null;
  const lastFalseExecution = executions.length > 0 ? executions.find((e) => e.outputs?.["false"]) : null;

  const processExecutionEventData = (execution: WorkflowsWorkflowNodeExecution): IfEvent => {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    const eventData: IfEvent = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: getState(componentName)(execution),
    };

    return eventData;
  };

  let trueEvent: IfEvent = {
    eventTitle: "No events received yet",
    eventState: "neutral" as const,
  };
  let falseEvent: IfEvent = {
    eventTitle: "No events received yet",
    eventState: "neutral" as const,
  };
  if (lastTrueExecution) {
    trueEvent = processExecutionEventData(lastTrueExecution!);
  }

  if (lastFalseExecution) {
    falseEvent = processExecutionEventData(lastFalseExecution!);
  }

  const eventSections: EventSection[] = [];
  if (trueEvent) {
    eventSections.push({
      title: "TRUE",
      ...trueEvent,
    });
  }
  if (falseEvent) {
    eventSections.push({
      title: "FALSE",
      ...falseEvent,
    });
  }

  return eventSections;
}

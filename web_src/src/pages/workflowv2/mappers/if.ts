import { ComponentsComponent, ComponentsNode, WorkflowsWorkflowNodeExecution, WorkflowsWorkflowNodeQueueItem } from "@/api-client";
import { ComponentBaseMapper } from "./types";
import { ComponentBaseProps, EventSection, EventState } from "@/ui/componentBase";
import { getTriggerRenderer } from ".";
import { getBackgroundColorClass } from "@/utils/colors";
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
    _componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _nodeQueueItems: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const expression = node.configuration?.expression as string | undefined;
    const conditions = expression ? parseExpression(expression) : [];
    const specs = expression
      ? [{
          title: "condition",
          tooltipTitle: "conditions applied",
          values: conditions,
        }]
      : undefined;

    return {
      iconSlug: "split",
      headerColor: "bg-gray-50",
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: getEventSections(nodes, lastExecutions),
      specs: specs,
      runDisabled: false,
      runDisabledTooltip: undefined,
    };
  },
};

function getEventSections(nodes: ComponentsNode[], executions: WorkflowsWorkflowNodeExecution[]): EventSection[] {
  const lastTrueExecution = executions.length > 0 ? executions.find((e) => e.outputs?.["true"]) : null;
  const lastFalseExecution = executions.length > 0 ? executions.find((e) => e.outputs?.["false"]) : null;

  const processExecutionEventData = (execution: WorkflowsWorkflowNodeExecution): IfEvent => {
    const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
    const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.trigger?.name || "");

    const { title } = rootTriggerRenderer.getTitleAndSubtitle(execution.rootEvent!);

    const eventData: IfEvent = {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventState: executionToEventSectionState(execution),
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

function executionToEventSectionState(execution: WorkflowsWorkflowNodeExecution): EventState {
  if (execution.state == "STATE_PENDING" || execution.state == "STATE_STARTED") {
    return "running";
  }

  if (execution.state == "STATE_FINISHED" && execution.result == "RESULT_PASSED") {
    return "success";
  }

  return "failed";
}

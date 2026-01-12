import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, OutputPayload, StateFunction } from "./types";
import { ComponentBaseProps, EventSection, EventState, EventStateMap, DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { parseExpression } from "@/lib/expressionParser";

type IfOutputs = Record<string, OutputPayload[]>;

export const IF_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  true: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  false: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
};

export const ifStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
  ) {
    return "error";
  }

  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }

  if (execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED") {
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const outputs = execution.outputs as IfOutputs | undefined;
    const trueOutputs = outputs?.true;
    if (Array.isArray(trueOutputs) && trueOutputs.length > 0) {
      return "true";
    }
    const falseOutputs = outputs?.false;
    if (Array.isArray(falseOutputs) && falseOutputs.length > 0) {
      return "false";
    }
    return "false";
  }

  return "failed";
};

export const IF_STATE_REGISTRY: EventStateRegistry = {
  stateMap: IF_STATE_MAP,
  getState: ifStateFunction,
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

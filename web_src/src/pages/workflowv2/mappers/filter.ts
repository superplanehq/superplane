import {
  ComponentsComponent,
  ComponentsNode,
  WorkflowsWorkflowNodeExecution,
  WorkflowsWorkflowNodeQueueItem,
} from "@/api-client";
import { ComponentBaseMapper, EventStateRegistry, OutputPayload, StateFunction } from "./types";
import {
  ComponentBaseProps,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { getBackgroundColorClass } from "@/utils/colors";
import { parseExpression, substituteExpressionValues, evaluateIndividualComparisons } from "@/lib/expressionParser";
import { formatTimeAgo } from "@/utils/date";

type FilterOutputs = Record<string, OutputPayload[]>;

export const FILTER_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  passed: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  rejected: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
};

export const filterStateFunction: StateFunction = (execution: WorkflowsWorkflowNodeExecution): EventState => {
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
    const outputs = execution.outputs as FilterOutputs | undefined;
    const hasOutputs = outputs
      ? Object.values(outputs).some((payloads) => Array.isArray(payloads) && payloads.length > 0)
      : false;
    return hasOutputs ? "passed" : "rejected";
  }

  return "failed";
};

export const FILTER_STATE_REGISTRY: EventStateRegistry = {
  stateMap: FILTER_STATE_MAP,
  getState: filterStateFunction,
};

export const filterMapper: ComponentBaseMapper = {
  props(
    nodes: ComponentsNode[],
    node: ComponentsNode,
    componentDefinition: ComponentsComponent,
    lastExecutions: WorkflowsWorkflowNodeExecution[],
    _queueItems: WorkflowsWorkflowNodeQueueItem[],
  ): ComponentBaseProps {
    const componentName = componentDefinition.name || "filter";
    // Prefer expression from execution metadata (stored at execution time)
    // Fall back to node configuration if metadata is not available
    const lastExecution = lastExecutions.length > 0 ? lastExecutions[0] : null;
    const metadata = lastExecution?.metadata as Record<string, any> | undefined;
    const expression = (metadata?.expression as string) || (node.configuration?.expression as string) || "";
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
      collapsed: node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: node.name!,
      eventSections: lastExecution ? getfilterEventSections(nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs,
      eventStateMap: getStateMap(componentName),
    };
  },
  subtitle(_node, execution) {
    if (!execution?.createdAt) return "";
    return formatTimeAgo(new Date(execution.createdAt));
  },
  getExecutionDetails(execution: WorkflowsWorkflowNodeExecution, node: ComponentsNode): Record<string, any> {
    const details: Record<string, any> = {};

    // Evaluated at
    if (execution.createdAt) {
      const evaluatedAt = new Date(execution.createdAt);
      details["Evaluated at"] = evaluatedAt.toLocaleString("en-US", {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false,
      });
    } else {
      details["Evaluated at"] = "N/A";
    }

    // Get the expression from execution metadata (stored at execution time)
    // Fall back to node configuration or execution configuration if metadata is not available
    const metadata = execution.metadata as Record<string, any> | undefined;
    const expression =
      (metadata?.expression as string) ||
      (node.configuration?.expression as string) ||
      (execution.configuration?.expression as string) ||
      "";

    // Evaluation (with values replaced) - formatted with badges and color-coded
    if (expression) {
      // Get the input data (payload) that was evaluated
      // Try execution.input first, then fall back to rootEvent.data
      let inputData: any = null;

      if (execution.input) {
        // Input might be an object directly or nested
        inputData = execution.input;
      } else if (execution.rootEvent?.data) {
        inputData = execution.rootEvent.data;
      }

      // Substitute values in the expression
      if (inputData) {
        const substitutedExpression = substituteExpressionValues(expression, inputData);
        const parsedEvaluation = parseExpression(substitutedExpression);

        // Determine if the filter passed (has outputs) or failed (no outputs)
        const outputs = execution.outputs as FilterOutputs | undefined;
        const hasOutputs = outputs
          ? Object.values(outputs).some((payloads) => Array.isArray(payloads) && payloads.length > 0)
          : false;
        const passed = hasOutputs;

        // Evaluate individual comparisons to determine which parts should be red
        const failedParts = evaluateIndividualComparisons(substitutedExpression);

        details["Evaluation"] = {
          __type: "evaluationBadges",
          values: parsedEvaluation,
          passed,
          failedParts: Array.from(failedParts),
        };
      } else {
        // If no input data available, show the expression as-is with badges
        const parsedExpression = parseExpression(expression);
        const outputs = execution.outputs as FilterOutputs | undefined;
        const hasOutputs = outputs
          ? Object.values(outputs).some((payloads) => Array.isArray(payloads) && payloads.length > 0)
          : false;
        const passed = hasOutputs;

        details["Evaluation"] = {
          __type: "evaluationBadges",
          values: parsedExpression,
          passed,
        };
      }
    } else {
      details["Evaluation"] = "N/A";
    }

    // Error (if present) - placed at the end, after Evaluation
    if (
      execution.resultMessage &&
      (execution.resultReason === "RESULT_REASON_ERROR" || execution.result === "RESULT_FAILED")
    ) {
      details["Error"] = {
        __type: "error",
        message: execution.resultMessage,
      };
    }

    return details;
  },
};

function getfilterEventSections(
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
    eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
    eventState: getState(componentName)(execution),
    eventId: execution.rootEvent?.id,
  };

  return [eventSection];
}

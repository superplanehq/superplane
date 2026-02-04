import { ComponentBaseContext, ComponentBaseMapper, EventStateRegistry, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload, StateFunction, SubtitleContext } from "./types";
import {
  ComponentBaseProps,
  EventSection,
  EventState,
  EventStateMap,
  DEFAULT_EVENT_STATE_MAP,
} from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import { parseExpression, substituteExpressionValues, evaluateIndividualComparisons } from "@/lib/expressionParser";
import { formatTimeAgo } from "@/utils/date";

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

export const ifStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
  if (!execution) return "neutral";

  if (
    execution.resultMessage &&
    (execution.resultReason === "RESULT_REASON_ERROR" ||
      (execution.result === "RESULT_FAILED" && execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
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

type IfConfiguration = {
  expression: string;
};

export const ifMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {  
    const componentName = context.componentDefinition.name || "if";
    const configuration = context.node.configuration as IfConfiguration;

    // Prefer expression from execution metadata (stored at execution time)
    // Fall back to node configuration if metadata is not available
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const metadata = lastExecution?.metadata as Record<string, any> | undefined;
    const expression = (metadata?.expression as string) || (configuration.expression as string) || undefined;
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
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component",
      eventSections: lastExecution ? getEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs: specs,
      runDisabled: false,
      runDisabledTooltip: undefined,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string {
    if (!context.execution.createdAt) return "";
    return formatTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const details: Record<string, any> = {};
    const configuration = context.node.configuration as IfConfiguration;

    // Evaluated at
    if (context.execution.createdAt) {
      const evaluatedAt = new Date(context.execution.createdAt);
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
    const metadata = context.execution.metadata as Record<string, any> | undefined;
    const expression =
      (metadata?.expression as string) ||
      (configuration.expression as string) ||
      (context.execution.configuration?.expression as string) ||
      "";

    // Evaluation (with values replaced) - formatted with badges and color-coded
    if (expression) {
      // Get the input data (payload) that was evaluated
      // Try execution.input first, then fall back to rootEvent.data
      let inputData: any = null;

      if (context.execution.input) {
        // Input might be an object directly or nested
        inputData = context.execution.input;
      } else if (context.execution.rootEvent?.data) {
        inputData = context.execution.rootEvent.data;
      }

      // Substitute values in the expression
      if (inputData) {
        const substitutedExpression = substituteExpressionValues(expression, inputData, {
          root: context.execution.rootEvent?.data,
          previousByDepth: { "1": inputData },
        });
        const parsedEvaluation = parseExpression(substitutedExpression);

        // Determine if the if condition evaluated to true (has outputs on "true" channel)
        const outputs = context.execution.outputs as IfOutputs | undefined;
        const trueOutputs = outputs?.true;
        const hasTrueOutputs = Array.isArray(trueOutputs) && trueOutputs.length > 0;
        const passed = hasTrueOutputs;

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
        const outputs = context.execution.outputs as IfOutputs | undefined;
        const trueOutputs = outputs?.true;
        const hasTrueOutputs = Array.isArray(trueOutputs) && trueOutputs.length > 0;
        const passed = hasTrueOutputs;

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
      context.execution.resultMessage &&
      (context.execution.resultReason === "RESULT_REASON_ERROR" ||
        (context.execution.result === "RESULT_FAILED" && context.execution.resultReason !== "RESULT_REASON_ERROR_RESOLVED"))
    ) {
      details["Error"] = {
        __type: "error",
        message: context.execution.resultMessage,
      };
    }

    return details;
  },
};

function getEventSections(
  nodes: NodeInfo[],
  execution: ExecutionInfo,
  componentName: string,
): EventSection[] {
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName!);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });

  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventTitle: title,
    eventSubtitle: formatTimeAgo(new Date(execution.createdAt!)),
    eventState: getState(componentName)(execution),
    eventId: execution.rootEvent!.id!,
  };

  return [eventSection];
}

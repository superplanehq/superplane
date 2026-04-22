import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "./types";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/pages/workflowv2/mappers/types";
import { DEFAULT_EVENT_STATE_MAP } from "@/pages/workflowv2/mappers/types";
import { getState, getStateMap } from ".";
import type React from "react";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";

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
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const specs = configuration.expression
      ? [
          {
            title: "Expression",
            tooltipTitle: "Expression",
            value: configuration.expression,
          },
        ]
      : undefined;

    return {
      iconSlug: "split",
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? getEventSections(lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs: specs,
      runDisabled: false,
      runDisabledTooltip: undefined,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const configuration = context.execution.configuration as IfConfiguration;
    const details: Record<string, any> = {
      "Evaluated at": context.execution.createdAt ? formatTimestampInUserTimezone(context.execution.createdAt) : "-",
      Expression: configuration.expression,
    };

    return details;
  },
};

function getEventSections(execution: ExecutionInfo, componentName: string): EventSection[] {
  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
    eventState: getState(componentName)(execution),
    eventId: execution.rootEvent!.id!,
  };

  return [eventSection];
}

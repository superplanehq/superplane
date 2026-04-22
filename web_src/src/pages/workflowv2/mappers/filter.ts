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
import { getBackgroundColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";

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

export const filterStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
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

type FilterConfiguration = {
  expression: string;
};

export const filterMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "filter";
    const configuration = context.node.configuration as FilterConfiguration;
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const specs = configuration.expression
      ? [
          {
            title: "Expression",
            tooltipTitle: "Filter expression",
            value: configuration.expression,
          },
        ]
      : undefined;

    return {
      iconSlug: "filter",
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title:
        context.node.name ||
        context.componentDefinition.label ||
        context.componentDefinition.name ||
        "Unnamed component",
      eventSections: lastExecution ? getfilterEventSections(lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, any> {
    const configuration = context.execution.configuration as FilterConfiguration;
    const details: Record<string, any> = {
      "Evaluated at": context.execution.createdAt ? formatTimestampInUserTimezone(context.execution.createdAt) : "-",
      Expression: configuration.expression,
    };

    return details;
  },
};

function getfilterEventSections(execution: ExecutionInfo, componentName: string): EventSection[] {
  const eventSection: EventSection = {
    receivedAt: new Date(execution.createdAt!),
    eventSubtitle: renderTimeAgo(new Date(execution.createdAt!)),
    eventState: getState(componentName)(execution),
    eventId: execution.rootEvent!.id!,
  };

  return [eventSection];
}

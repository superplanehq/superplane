import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  StateFunction,
  SubtitleContext,
} from "./types";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { getTriggerRenderer, getState, getStateMap } from ".";
import type React from "react";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";
import { renderTimeAgo } from "@/components/TimeAgo";
import { formatTimestampInUserTimezone } from "@/lib/timezone";

type ForEachOutputs = Record<string, OutputPayload[]>;

export const FOR_EACH_STATE_MAP: EventStateMap = {
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

export const forEachStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
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
    const outputs = execution.outputs as ForEachOutputs | undefined;
    const itemOutputs = outputs?.item;
    const hasItems = Array.isArray(itemOutputs) && itemOutputs.length > 0;
    return hasItems ? "passed" : "rejected";
  }

  return "failed";
};

export const FOR_EACH_STATE_REGISTRY: EventStateRegistry = {
  stateMap: FOR_EACH_STATE_MAP,
  getState: forEachStateFunction,
};

type ForEachConfiguration = {
  arrayExpression: string;
};

type ForEachMetadata = {
  arrayExpression?: string;
  count?: number;
};

export const forEachMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "forEach";
    const configuration = context.node.configuration as ForEachConfiguration;
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const specs = configuration.arrayExpression
      ? [
          {
            title: "Array",
            tooltipTitle: "Array expression",
            value: configuration.arrayExpression,
          },
        ]
      : undefined;

    return {
      iconSlug: context.componentDefinition.icon ?? "repeat",
      iconColor: getColorClass(context.componentDefinition.color ?? "blue"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "For Each",
      eventSections: lastExecution ? getForEachEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string | number> {
    const configuration = context.execution.configuration as ForEachConfiguration;
    const metadata = context.execution.metadata as ForEachMetadata | undefined;
    const details: Record<string, string | number> = {
      "Evaluated at": context.execution.createdAt ? formatTimestampInUserTimezone(context.execution.createdAt) : "-",
      "Array expression": configuration.arrayExpression ?? metadata?.arrayExpression ?? "-",
    };

    if (typeof metadata?.count === "number") {
      details["Items emitted"] = metadata.count;
    }

    return details;
  },
};

function getForEachEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
  const rootEvent = execution.rootEvent;
  if (!rootEvent?.id || !execution.createdAt) {
    return [];
  }

  const rootTriggerNode = nodes.find((n) => n.id === rootEvent.nodeId);
  if (!rootTriggerNode?.componentName) {
    return [];
  }

  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode.componentName);
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: rootEvent });
  const createdAt = new Date(execution.createdAt);

  return [
    {
      receivedAt: createdAt,
      eventTitle: title,
      eventSubtitle: renderTimeAgo(createdAt),
      eventState: getState(componentName)(execution),
      eventId: rootEvent.id,
    },
  ];
}

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

type LoopOutputs = Record<string, OutputPayload[]>;

export const LOOP_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  body: {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-indigo-100",
    badgeColor: "bg-indigo-500",
  },
  done: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
};

export const loopStateFunction: StateFunction = (execution: ExecutionInfo): EventState => {
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
    const outputs = execution.outputs as LoopOutputs | undefined;
    if (Array.isArray(outputs?.done) && outputs.done.length > 0) {
      return "done";
    }
    if (Array.isArray(outputs?.body) && outputs.body.length > 0) {
      return "body";
    }
  }

  return "failed";
};

export const LOOP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: LOOP_STATE_MAP,
  getState: loopStateFunction,
};

type LoopConfiguration = {
  untilExpression?: string;
  maxIterations?: number;
};

type LoopMetadata = {
  iteration?: number;
  active?: boolean;
};

export const loopMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "loop";
    const configuration = context.node.configuration as LoopConfiguration;
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const specs = configuration.untilExpression
      ? [
          {
            title: "Until",
            tooltipTitle: "Until expression",
            value: configuration.untilExpression,
          },
        ]
      : undefined;

    return {
      iconSlug: context.componentDefinition.icon ?? "refresh-cw",
      iconColor: getColorClass(context.componentDefinition.color ?? "indigo"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Loop",
      eventSections: lastExecution ? getLoopEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      specs,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string | number | boolean> {
    const configuration = context.execution.configuration as LoopConfiguration;
    const metadata = context.execution.metadata as LoopMetadata | undefined;
    const details: Record<string, string | number | boolean> = {
      "Evaluated at": context.execution.createdAt ? formatTimestampInUserTimezone(context.execution.createdAt) : "-",
      "Until expression": configuration.untilExpression ?? "-",
      "Max iterations": configuration.maxIterations ?? 100,
    };

    if (typeof metadata?.iteration === "number") {
      details["Current iteration"] = metadata.iteration;
    }
    if (typeof metadata?.active === "boolean") {
      details.Active = metadata.active;
    }

    return details;
  },
};

function getLoopEventSections(nodes: NodeInfo[], execution: ExecutionInfo, componentName: string): EventSection[] {
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

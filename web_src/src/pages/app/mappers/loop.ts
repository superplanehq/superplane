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
    const iterationOutputs = outputs?.iteration;
    const hasIterations = Array.isArray(iterationOutputs) && iterationOutputs.length > 0;
    return hasIterations ? "passed" : "rejected";
  }

  return "failed";
};

export const LOOP_STATE_REGISTRY: EventStateRegistry = {
  stateMap: LOOP_STATE_MAP,
  getState: loopStateFunction,
};

type LoopConfiguration = {
  mode?: "collection" | "count" | "range";
  collectionExpression?: string;
  countExpression?: string;
  startExpression?: string;
  endExpression?: string;
  stepExpression?: string;
  itemVariable?: string;
  payloadExpression?: string;
};

type LoopMetadata = {
  mode?: string;
  count?: number;
  itemVariable?: string;
};

const MODE_LABELS: Record<string, string> = {
  collection: "Collection",
  count: "Count",
  range: "Range",
};

function getLoopSummary(configuration: LoopConfiguration): string | undefined {
  switch (configuration.mode ?? "collection") {
    case "collection":
      return configuration.collectionExpression;
    case "count":
      return configuration.countExpression;
    case "range": {
      const parts = [configuration.startExpression, configuration.endExpression].filter(Boolean);
      if (configuration.stepExpression) {
        parts.push(`step ${configuration.stepExpression}`);
      }
      return parts.length > 0 ? parts.join(" → ") : undefined;
    }
    default:
      return undefined;
  }
}

export const loopMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "loop";
    const configuration = context.node.configuration as LoopConfiguration;
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const mode = configuration.mode ?? "collection";
    const summary = getLoopSummary(configuration);
    const specs = summary
      ? [
          {
            title: MODE_LABELS[mode] ?? "Loop",
            tooltipTitle: `${MODE_LABELS[mode] ?? "Loop"} configuration`,
            value: summary,
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

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string | number> {
    const configuration = context.execution.configuration as LoopConfiguration;
    const metadata = context.execution.metadata as LoopMetadata | undefined;
    const mode = configuration.mode ?? metadata?.mode ?? "collection";
    const details: Record<string, string | number> = {
      "Evaluated at": context.execution.createdAt ? formatTimestampInUserTimezone(context.execution.createdAt) : "-",
      Mode: MODE_LABELS[mode] ?? mode,
    };

    switch (mode) {
      case "collection":
        details["Collection expression"] = configuration.collectionExpression ?? "-";
        break;
      case "count":
        details["Count expression"] = configuration.countExpression ?? "-";
        break;
      case "range":
        details["Start expression"] = configuration.startExpression ?? "-";
        details["End expression"] = configuration.endExpression ?? "-";
        if (configuration.stepExpression) {
          details["Step expression"] = configuration.stepExpression;
        }
        break;
    }

    if (configuration.itemVariable) {
      details["Item variable"] = configuration.itemVariable;
    }
    if (configuration.payloadExpression) {
      details["Payload expression"] = configuration.payloadExpression;
    }
    if (typeof metadata?.count === "number") {
      details["Iterations emitted"] = metadata.count;
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

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
  done: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  next: {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-indigo-100",
    badgeColor: "bg-indigo-500",
  },
  waiting: {
    icon: "clock",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
};

function isWaitingBetweenIterations(execution: ExecutionInfo): boolean {
  if (execution.state !== "STATE_PENDING" && execution.state !== "STATE_STARTED") {
    return false;
  }

  const metadata = execution.metadata as LoopMetadata | undefined;
  return metadata?.waitingBetweenIterations === true;
}

function resolveFinishedLoopState(outputs: LoopOutputs | undefined): EventState | null {
  if (Array.isArray(outputs?.done) && outputs.done.length > 0) {
    return "done";
  }
  if (Array.isArray(outputs?.next) && outputs.next.length > 0) {
    return "next";
  }
  return null;
}

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
    if (isWaitingBetweenIterations(execution)) {
      return "waiting";
    }
    return "running";
  }

  if (execution.state === "STATE_FINISHED" && execution.result === "RESULT_PASSED") {
    const finishedState = resolveFinishedLoopState(execution.outputs as LoopOutputs | undefined);
    if (finishedState) {
      return finishedState;
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
  timeoutSeconds?: number;
  delayBetweenIterations?: DelaySpec;
};

type DelaySpec = {
  enabled?: boolean;
  strategy?: "fixed" | "exponential";
  intervalSeconds?: number;
};

type LoopMetadata = {
  iteration?: number;
  maxIterations?: number;
  active?: boolean;
  waitingBetweenIterations?: boolean;
};

export const loopMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const componentName = context.componentDefinition.name || "loop";
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;

    return {
      iconSlug: context.componentDefinition.icon ?? "refresh-cw",
      iconColor: getColorClass(context.componentDefinition.color ?? "indigo"),
      collapsed: context.node.isCollapsed,
      collapsedBackground: getBackgroundColorClass("white"),
      title: context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Loop",
      eventSections: lastExecution ? getLoopEventSections(context.nodes, lastExecution, componentName) : undefined,
      includeEmptyState: !lastExecution,
      eventStateMap: getStateMap(componentName),
    };
  },

  subtitle(context: SubtitleContext): string | React.ReactNode {
    if (!context.execution.createdAt) return "";
    return renderTimeAgo(new Date(context.execution.createdAt));
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string | number | boolean> {
    const configuration = context.execution.configuration as LoopConfiguration;
    const sessionMetadata = isLoopSessionMetadata(context.execution.metadata) ? context.execution.metadata : undefined;
    const details: Record<string, string | number | boolean> = {
      "Evaluated at": context.execution.createdAt ? formatTimestampInUserTimezone(context.execution.createdAt) : "-",
      "Until expression": configuration.untilExpression ?? "-",
      "Max iterations": configuration.maxIterations ?? 10,
      "Timeout (s)": configuration.timeoutSeconds ?? 3600,
    };

    if (typeof sessionMetadata?.iteration === "number") {
      details["Current iteration"] = sessionMetadata.iteration;
    }
    if (typeof sessionMetadata?.active === "boolean") {
      details.Active = sessionMetadata.active;
    }
    if (sessionMetadata?.waitingBetweenIterations) {
      details["Waiting between iterations"] = true;
    }

    const doneOutput = (context.execution.outputs as LoopOutputs | undefined)?.done?.[0]?.data as
      | {
          done?: {
            iterations?: number;
            stopReason?: string;
            elapsedMs?: number;
          };
        }
      | undefined;
    if (typeof doneOutput?.done?.iterations === "number") {
      details.Iterations = doneOutput.done.iterations;
    }
    if (doneOutput?.done?.stopReason) {
      details["Stop reason"] = doneOutput.done.stopReason;
    }
    if (typeof doneOutput?.done?.elapsedMs === "number") {
      details["Elapsed (ms)"] = doneOutput.done.elapsedMs;
    }

    if (configuration.delayBetweenIterations?.enabled) {
      const strategy = configuration.delayBetweenIterations.strategy ?? "fixed";
      const interval = configuration.delayBetweenIterations.intervalSeconds ?? 0;
      details["Delay strategy"] = strategy;
      details["Delay interval (s)"] = interval;
    }

    return details;
  },
};

function isLoopSessionMetadata(metadata: unknown): metadata is LoopMetadata {
  return typeof metadata === "object" && metadata !== null && "active" in metadata;
}

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

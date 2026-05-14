import { renderTimeAgo } from "@/components/TimeAgo";
import { getColorClass } from "@/lib/colors";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
import { RunnerLiveLogDialog } from "@/ui/CanvasPage/RunnerLiveLogDialog";
import React from "react";
import { getTriggerRenderer } from ".";

import type {
  ComponentBaseContext,
  ComponentBaseMapper,
  EventStateRegistry,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
} from "./types";

import { stringOrDash } from "./utils";

const RUNNER_STATE_MAP: EventStateMap = {
  ...DEFAULT_EVENT_STATE_MAP,
  passed: DEFAULT_EVENT_STATE_MAP.success,
};

type RunnerOutputs = { passed?: OutputPayload[]; failed?: OutputPayload[]; default?: OutputPayload[] } | undefined;

function firstRunnerPayload(execution: ExecutionInfo): Record<string, unknown> | undefined {
  const outputs = execution.outputs as RunnerOutputs;
  const payload = outputs?.failed?.[0]?.data ?? outputs?.passed?.[0]?.data ?? outputs?.default?.[0]?.data;
  if (!payload || typeof payload !== "object") return undefined;
  return payload as Record<string, unknown>;
}

function runnerFinishedPassedState(execution: ExecutionInfo): EventState {
  const outputs = execution.outputs as RunnerOutputs;
  if (outputs?.failed?.length) {
    return "failed";
  }
  if (outputs?.passed?.length) {
    return "passed";
  }

  const payload = firstRunnerPayload(execution);
  const exitCode = payload?.exit_code;
  if (typeof exitCode === "number") {
    return exitCode === 0 ? "passed" : "failed";
  }
  return "passed";
}

const runnerStateFunction = (execution: ExecutionInfo): EventState => {
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
    return runnerFinishedPassedState(execution);
  }

  return "failed";
};

export const RUNNER_STATE_REGISTRY: EventStateRegistry = {
  stateMap: RUNNER_STATE_MAP,
  getState: runnerStateFunction,
};

export const runnerMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const lastExecution = context.lastExecutions.length > 0 ? context.lastExecutions[0] : null;
    const title =
      context.node.name || context.componentDefinition.label || context.componentDefinition.name || "Unnamed component";
    const iconSlug = context.componentDefinition.icon || "terminal";
    const iconColor = getColorClass(context.componentDefinition?.color || "blue");

    const customField =
      lastExecution && context.canvasMode === "live"
        ? () => <RunnerLiveLogDialog canvasMode={context.canvasMode ?? "live"} executionId={lastExecution.id} />
        : undefined;

    return {
      title,
      iconSlug,
      iconColor,
      collapsed: context.node.isCollapsed,
      collapsedBackground: "bg-white",
      eventSections: lastExecution ? runnerEventSections(context.nodes, lastExecution) : undefined,
      includeEmptyState: !lastExecution,
      metadata: [],
      specs: [],
      eventStateMap: RUNNER_STATE_MAP,
      customField,
      customFieldPosition: "before",
    };
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {};
    const payload = firstRunnerPayload(context.execution);
    if (!payload) return details;

    details["status"] = stringOrDash(payload.status);
    details["exit_code"] = stringOrDash(payload.exit_code);
    return details;
  },
};

function runnerEventSections(nodes: NodeInfo[], execution: ExecutionInfo): EventSection[] | undefined {
  if (!execution) return undefined;
  const rootTriggerNode = nodes.find((n) => n.id === execution.rootEvent?.nodeId);
  const rootTriggerRenderer = getTriggerRenderer(rootTriggerNode?.componentName ?? "");
  const { title } = rootTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent });
  const state = runnerStateFunction(execution);
  const subtitleTimestamp = state === "running" ? execution.createdAt : execution.updatedAt || execution.createdAt;
  const eventSubtitle = subtitleTimestamp ? renderTimeAgo(new Date(subtitleTimestamp)) : undefined;
  return [
    {
      receivedAt: new Date(execution.createdAt!),
      eventTitle: title,
      eventSubtitle,
      eventState: state,
      eventId: execution.rootEvent!.id!,
    },
  ];
}

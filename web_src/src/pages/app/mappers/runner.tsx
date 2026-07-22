import { renderTimeAgo } from "@/components/TimeAgo";
import { getColorClass } from "@/lib/colors";
import { RunnerLiveLogDialog } from "@/ui/CanvasPage/RunnerLiveLogDialog";
import type { ComponentBaseProps, EventSection, EventState, EventStateMap } from "@/ui/componentBase";
import { DEFAULT_EVENT_STATE_MAP } from "@/ui/componentBase";
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

const DEFAULT_EXECUTION_TIMEOUT_SECONDS = 3600;
const BROKER_TASK_ID_METADATA_KEY = "runner_broker_task_id";

const EXECUTION_MODE_DOCKER = "docker";
const DOCKER_IMAGE_PRESET_CUSTOM = "custom";

/** Mirrors `resolvedDockerImageRef` in pkg/components/runner/spec.go for execution summaries. */
function resolvedContainerImageRef(c: Record<string, unknown>): string {
  const rawMode = typeof c.execution_mode === "string" ? c.execution_mode.trim().toLowerCase() : "";
  if (rawMode !== EXECUTION_MODE_DOCKER) {
    return "";
  }
  const preset = typeof c.docker_image_preset === "string" ? c.docker_image_preset.trim() : "";
  const custom = typeof c.docker_image === "string" ? c.docker_image.trim() : "";
  if (!preset) {
    return custom;
  }
  if (preset === DOCKER_IMAGE_PRESET_CUSTOM) {
    return custom;
  }
  return preset;
}

/** Exported for tests; mirrors runner node configuration shown in execution details. */
export function runnerConfigurationDetails(configuration: unknown): Record<string, string> {
  const details: Record<string, string> = {};
  if (!configuration || typeof configuration !== "object") {
    return details;
  }
  const c = configuration as Record<string, unknown>;
  const machineTypeRaw = c.machineType ?? c.machine_type;
  const machineType = typeof machineTypeRaw === "string" ? machineTypeRaw.trim() : "";
  if (machineType) {
    details["Machine type"] = machineType;
  }
  const rawMode = typeof c.execution_mode === "string" ? c.execution_mode.trim().toLowerCase() : "";
  if (rawMode === EXECUTION_MODE_DOCKER) {
    details["Execution mode"] = "Docker";
  } else {
    details["Execution mode"] = "Host";
  }
  const image = resolvedContainerImageRef(c);
  if (image) {
    details["Container image"] = image;
  }
  const timeoutRaw = c.executionTimeoutSeconds ?? c.execution_timeout_seconds;
  const timeoutLabel = (value: number | string) => {
    const parsed = typeof value === "number" ? value : Number.parseInt(value.trim(), 10);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return String(DEFAULT_EXECUTION_TIMEOUT_SECONDS);
    }
    return String(Math.trunc(parsed));
  };
  if (typeof timeoutRaw === "number" && Number.isFinite(timeoutRaw)) {
    details["Timeout (seconds)"] = timeoutLabel(timeoutRaw);
  } else if (typeof timeoutRaw === "string" && timeoutRaw.trim() !== "") {
    details["Timeout (seconds)"] = timeoutLabel(timeoutRaw);
  } else {
    details["Timeout (seconds)"] = String(DEFAULT_EXECUTION_TIMEOUT_SECONDS);
  }
  return details;
}

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

function brokerTaskIDFromExecution(execution: ExecutionInfo): string | undefined {
  const meta = execution.metadata;
  if (meta && typeof meta === "object") {
    const id = (meta as Record<string, unknown>)[BROKER_TASK_ID_METADATA_KEY];
    if (typeof id === "string" && id.trim() !== "") {
      return id.trim();
    }
  }

  const payload = firstRunnerPayload(execution);
  const taskID = payload?.task_id;
  if (typeof taskID === "string" && taskID.trim() !== "") {
    return taskID.trim();
  }

  return undefined;
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

  if (execution.state === "STATE_CANCELLING") {
    return "cancelling";
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
    const componentDef = context.componentDefinition;
    const title = context.node.name || componentDef.label || componentDef.name || "Unnamed component";
    const iconSlug = context.componentDefinition.icon || "terminal";
    const iconColor = getColorClass(context.componentDefinition?.color || "blue");
    const canvasMode = context.canvasMode ?? "live";

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
      customField: <RunnerLiveLogDialog title={title} canvasMode={canvasMode} execution={lastExecution} />,
      customFieldPosition: "after",
    };
  },
  subtitle(context: SubtitleContext): string | React.ReactNode {
    const timestamp = context.execution.updatedAt || context.execution.createdAt;
    return timestamp ? renderTimeAgo(new Date(timestamp)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const details: Record<string, string> = {
      ...runnerConfigurationDetails(context.node.configuration),
    };

    const taskID = brokerTaskIDFromExecution(context.execution);
    if (taskID) {
      details["task_id"] = taskID;
    }

    const payload = firstRunnerPayload(context.execution);
    if (!payload) {
      return details;
    }

    details["Status"] = stringOrDash(payload.status);
    details["Exit code"] = stringOrDash(payload.exit_code);
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

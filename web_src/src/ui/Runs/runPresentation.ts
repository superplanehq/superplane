/* eslint-disable complexity */
import type {
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasRun,
  CanvasesCanvasRunResult,
  CanvasesCanvasRunState,
  SuperplaneComponentsNode,
} from "@/api-client";
import { getTriggerRenderer } from "@/pages/app/mappers";
import { buildEventInfo } from "@/pages/app/utils";
import { AlertTriangle, CheckCircle2, CircleDashed, Clock, MinusCircle, type LucideIcon } from "lucide-react";

import { RUN_STATUS_FILTER_IDS, type RunStatusFilter } from "./runStatusFilterVocab";

export type { RunStatusFilter };
export type RunResultFilter = Exclude<RunStatusFilter, "running">;
export type RunStatusKey = RunStatusFilter | "unknown";

const RUN_STATUS_FILTER_OPTION_META: Record<RunStatusFilter, { label: string; dotClassName: string }> = {
  running: { label: "Running", dotClassName: "bg-blue-500" },
  passed: { label: "Passed", dotClassName: "bg-emerald-500" },
  failed: { label: "Failed", dotClassName: "bg-red-500" },
  cancelled: { label: "Cancelled", dotClassName: "bg-gray-400" },
};

export const RUN_STATUS_FILTER_OPTIONS: { id: RunStatusFilter; label: string; dotClassName: string }[] =
  RUN_STATUS_FILTER_IDS.map((id) => ({ id, ...RUN_STATUS_FILTER_OPTION_META[id] }));

export const RUN_STATUS_META = {
  running: {
    label: "Running",
    badgeClassName: "bg-blue-100 text-blue-700 dark:bg-blue-950/70 dark:text-blue-300",
    dotClassName: "bg-blue-500 animate-pulse",
    icon: Clock,
  },
  failed: {
    label: "Failed",
    badgeClassName: "bg-red-100 text-red-700 dark:bg-red-950/70 dark:text-red-300",
    dotClassName: "bg-red-500",
    icon: AlertTriangle,
  },
  cancelled: {
    label: "Cancelled",
    badgeClassName: "bg-slate-200 text-gray-700 dark:bg-slate-900 dark:text-gray-300",
    dotClassName: "bg-gray-400",
    icon: MinusCircle,
  },
  passed: {
    label: "Passed",
    badgeClassName: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/70 dark:text-emerald-300",
    dotClassName: "bg-emerald-500",
    icon: CheckCircle2,
  },
  unknown: {
    label: "Unknown",
    badgeClassName: "bg-slate-200 text-slate-600 dark:bg-slate-900 dark:text-slate-300",
    dotClassName: "bg-slate-300",
    icon: CircleDashed,
  },
} satisfies Record<
  RunStatusKey,
  {
    label: string;
    badgeClassName: string;
    dotClassName: string;
    icon: LucideIcon;
  }
>;

export function shortId(value: string | undefined) {
  return value ? value.slice(0, 8) : "";
}

export function statusFiltersToApiFilters(filters: RunStatusFilter[]): {
  states: CanvasesCanvasRunState[];
  results: CanvasesCanvasRunResult[];
} {
  const resultByFilter: Record<RunResultFilter, CanvasesCanvasRunResult> = {
    passed: "RESULT_PASSED",
    failed: "RESULT_FAILED",
    cancelled: "RESULT_CANCELLED",
  };

  const states: CanvasesCanvasRunState[] = filters.includes("running") ? ["STATE_STARTED"] : [];
  const results = filters
    .filter((filter): filter is RunResultFilter => filter !== "running")
    .map((filter) => resultByFilter[filter]);

  return { states, results };
}

export function getRunStatus(run: CanvasesCanvasRun): RunStatusKey {
  if (run.state === "STATE_STARTED") return "running";
  if (run.result === "RESULT_FAILED") return "failed";
  if (run.result === "RESULT_CANCELLED") return "cancelled";
  if (run.result === "RESULT_PASSED" || run.state === "STATE_FINISHED") return "passed";
  return "unknown";
}

function getExecutionStatusLabel(execution: CanvasesCanvasNodeExecutionRef) {
  if (execution.state === "STATE_PENDING") return "Pending";
  if (execution.state === "STATE_CANCELLING") return "Cancelling";
  if (execution.state === "STATE_STARTED") return "Running";
  if (execution.result === "RESULT_FAILED") return "Failed";
  if (execution.result === "RESULT_CANCELLED") return "Cancelled";
  if (execution.result === "RESULT_PASSED") return "Passed";
  return "Unknown";
}

export function getExecutionStatus(execution: CanvasesCanvasNodeExecutionRef) {
  const statusLabel = getExecutionStatusLabel(execution);

  if (
    execution.state === "STATE_STARTED" ||
    execution.state === "STATE_PENDING" ||
    execution.state === "STATE_CANCELLING"
  ) {
    return {
      label: statusLabel,
      className: "bg-blue-50 text-blue-700 ring-blue-200",
      dotClassName: "bg-blue-500",
    };
  }

  if (execution.result === "RESULT_FAILED") {
    return { label: statusLabel, className: "bg-red-50 text-red-700 ring-red-200", dotClassName: "bg-red-500" };
  }

  if (execution.result === "RESULT_CANCELLED") {
    return { label: statusLabel, className: "bg-gray-100 text-gray-700 ring-gray-200", dotClassName: "bg-gray-400" };
  }

  if (execution.result === "RESULT_PASSED" || execution.state === "STATE_FINISHED") {
    return {
      label: statusLabel,
      className: "bg-emerald-50 text-emerald-700 ring-emerald-200",
      dotClassName: "bg-emerald-500",
    };
  }

  return { label: statusLabel, className: "bg-slate-100 text-slate-600 ring-slate-200", dotClassName: "bg-slate-300" };
}

export function buildNodeMap(workflowNodes: SuperplaneComponentsNode[]) {
  const map = new Map<string, SuperplaneComponentsNode>();
  for (const node of workflowNodes) {
    if (node.id) map.set(node.id, node);
  }
  return map;
}

export function buildRunPresentation(run: CanvasesCanvasRun, nodeMap: Map<string, SuperplaneComponentsNode>) {
  const rootEvent = run.rootEvent;
  const triggerNode = rootEvent?.nodeId ? nodeMap.get(rootEvent.nodeId) : undefined;
  const triggerName = triggerNode?.name || triggerNode?.component || "Trigger";
  const eventInfo = rootEvent ? buildEventInfo(rootEvent) : null;
  const triggerRenderer = getTriggerRenderer(triggerNode?.component || "");
  const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "" };
  const status = getRunStatus(run);
  const displayTitle = title || rootEvent?.customName || `Run ${shortId(run.id)}`;

  return {
    run,
    rootEvent,
    triggerNode,
    triggerName,
    title: displayTitle,
    status,
    haystack: [displayTitle, triggerName, run.id, rootEvent?.id, rootEvent?.customName]
      .filter(Boolean)
      .join(" ")
      .toLowerCase(),
  };
}

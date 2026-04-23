import type {
  CanvasesCanvasEventWithExecutions,
  CanvasesCanvasNodeExecution,
  CanvasesCanvasNodeExecutionRef,
  CanvasesCanvasNodeQueueItem,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { formatDurationSeconds } from "@/lib/duration";
import { DEFAULT_EVENT_STATE_MAP, type EventState } from "@/ui/componentBase";
import { getState, getStateMap, getTriggerRenderer } from "../mappers";
import { defaultStateFunction } from "../mappers/stateRegistry";
import { buildEventInfo, buildExecutionInfo } from "../utils";

export function resolveNodeIconSlug(
  node: ComponentsNode | undefined,
  componentIconMap: Record<string, string>,
): string | undefined {
  const name = node?.component?.name || node?.trigger?.name;
  if (!name) return undefined;
  return componentIconMap[name];
}

export function getAggregateStatus(executions: CanvasesCanvasNodeExecutionRef[]) {
  if (executions.some((e) => e.state === "STATE_STARTED" || e.state === "STATE_PENDING")) {
    return "running";
  }
  if (executions.some((e) => e.result === "RESULT_FAILED")) {
    return "error";
  }
  if (executions.some((e) => e.result === "RESULT_CANCELLED")) {
    return "cancelled";
  }
  if (executions.every((e) => e.result === "RESULT_PASSED")) {
    return "completed";
  }
  if (executions.every((e) => e.state === "STATE_FINISHED")) {
    return "completed";
  }
  return "queued";
}

//
// Like getAggregateStatus but also factors in pending queue items for the
// run. A run that has every execution finished (thus "completed" by the
// execution-only logic) but still has components sitting in the queue is
// really a "queued" run -- it hasn't finished all its work.
//
// "running" / "error" / "cancelled" outcomes always take precedence so the
// user still sees the most severe state, consistent with getAggregateStatus.
//
export function getAggregateRunStatus(
  executions: CanvasesCanvasNodeExecutionRef[],
  hasPendingQueueItems: boolean,
): string {
  if (executions.length === 0) {
    return hasPendingQueueItems ? "queued" : "queued";
  }
  const base = getAggregateStatus(executions);
  if (hasPendingQueueItems && base === "completed") return "queued";
  return base;
}

export function computeDuration(execution: CanvasesCanvasNodeExecutionRef): string | null {
  if (execution.state !== "STATE_FINISHED" || !execution.createdAt || !execution.updatedAt) {
    return null;
  }
  const ms = new Date(execution.updatedAt).getTime() - new Date(execution.createdAt).getTime();
  return formatDurationSeconds(ms);
}

export function resolveExecutionDisplayStatus(execution: CanvasesCanvasNodeExecution, nodes: ComponentsNode[]): string {
  const node = nodes.find((n) => n.id === execution.nodeId);
  const componentName = node?.component?.name || "";
  const stateResolver = getState(componentName);
  const componentState = stateResolver(buildExecutionInfo(execution));

  if (componentState && componentState !== "neutral") return componentState;
  if (execution.state === "STATE_PENDING") return "pending";
  if (execution.state === "STATE_STARTED") return "running";
  if (execution.result === "RESULT_CANCELLED") return "cancelled";
  if (execution.result === "RESULT_FAILED") return "failed";
  if (execution.result === "RESULT_PASSED") return "success";
  return "unknown";
}

//
// Returns the canonical EventState for an execution, bypassing per-component
// state labels (e.g. "created" / "deleted"). Use this when you need a
// canvas-consistent color without caring about the custom label. Components
// without a specific mapper fall through to the default lifecycle logic in
// `defaultStateFunction`. This is the colors-by-state side of
// `resolveExecutionDisplayStatus`, which handles labels.
//
export function resolveExecutionEventState(execution: CanvasesCanvasNodeExecution): EventState {
  return defaultStateFunction(buildExecutionInfo(execution));
}

//
// Resolves the badge color (tailwind bg class) for an execution using the
// component's own EventStateMap -- same lookup the canvas node does. This
// picks up component-specific custom colors (e.g. wait's amber for
// "pushed through") that aren't expressible via the canonical EventState
// palette. Falls back to the canonical map via `defaultStateFunction` when
// the component doesn't define a specific style.
//
export function resolveExecutionBadgeColor(
  execution: CanvasesCanvasNodeExecution,
  nodes: ComponentsNode[],
): string {
  const node = nodes.find((n) => n.id === execution.nodeId);
  const componentName = node?.component?.name || "";
  const execInfo = buildExecutionInfo(execution);
  const stateResolver = getState(componentName);
  const stateMap = getStateMap(componentName);
  const customState = stateResolver(execInfo);
  const customStyle = stateMap[customState];
  if (customStyle?.badgeColor) return customStyle.badgeColor;

  const canonical = defaultStateFunction(execInfo);
  return (DEFAULT_EVENT_STATE_MAP[canonical] || DEFAULT_EVENT_STATE_MAP.neutral).badgeColor;
}

//
// Tailwind bg class for a canonical EventState, used for non-execution
// steps (trigger, synthetic queued). Executions should use
// `resolveExecutionBadgeColor` so they pick up component-specific palettes.
//
export function badgeColorForEventState(state: EventState): string {
  return (DEFAULT_EVENT_STATE_MAP[state] || DEFAULT_EVENT_STATE_MAP.neutral).badgeColor;
}

const STATUS_TO_EVENT_STATE: Record<string, EventState> = {
  completed: "success",
  success: "success",
  triggered: "triggered",
  finished: "success",
  passed: "success",
  approved: "success",
  opened: "success",
  true: "success",
  waiting: "queued",
  queued: "queued",
  pending: "queued",
  running: "running",
  failed: "failed",
  error: "error",
  rejected: "error",
  false: "error",
  timeout: "failed",
  cancelled: "cancelled",
  "pushed through": "running",
};

const STATUS_LABELS: Record<string, string> = {
  completed: "Completed",
  success: "Success",
  triggered: "Triggered",
  finished: "Finished",
  waiting: "Waiting",
  approved: "Approved",
  rejected: "Rejected",
  "pushed through": "Pushed Through",
  opened: "Opened",
  true: "True",
  false: "False",
  passed: "Passed",
  timeout: "Timeout",
  queued: "Queued",
  failed: "Failed",
  running: "Running",
  pending: "Pending",
  cancelled: "Cancelled",
  error: "Error",
};

//
// Resolves a raw status string (as emitted by resolveExecutionDisplayStatus,
// component state functions, or getAggregateStatus) to the canonical
// EventState that drives colors across the canvas surfaces (badges, node
// dots, ribbon bars, activity row accents). Unknown strings fall to
// "neutral" so they get a muted gray rather than being silently bucketed
// with another state.
//
export function resolveEventState(status: string): EventState {
  return STATUS_TO_EVENT_STATE[status] || "neutral";
}

//
// Title-cases an unknown status label so component-specific states
// (e.g. "created", "deleted", "power on") read nicely in UI pills instead
// of looking like raw enums. Known labels from STATUS_LABELS already have
// their own casing and take precedence.
//
function formatStatusLabel(status: string): string {
  if (!status) return "Unknown";
  const normalized = status.replace(/[_-]+/g, " ").trim();
  if (!normalized) return "Unknown";
  return normalized
    .split(/\s+/)
    .map((word) => (word.length > 0 ? word[0].toUpperCase() + word.slice(1) : word))
    .join(" ");
}

//
// Returns the visual props (badge color + label) for a raw status. Callers
// that already know the exact color (e.g. resolved via
// `resolveExecutionBadgeColor` against the component's own state map) can
// pass it as `badgeColorOverride` so the pill matches the canvas for
// component-specific colors like wait's amber "pushed through". If only
// the canonical `eventState` is known we use the canonical palette. As a
// last resort we fall back to mapping the raw status string.
//
export function getStatusBadgeProps(
  status: string,
  eventState?: EventState,
  badgeColorOverride?: string,
): { badgeColor: string; label: string } {
  const label = STATUS_LABELS[status] || formatStatusLabel(status);

  if (badgeColorOverride) {
    return { badgeColor: badgeColorOverride, label };
  }

  const resolved = eventState ?? resolveEventState(status);
  const style = DEFAULT_EVENT_STATE_MAP[resolved] || DEFAULT_EVENT_STATE_MAP.neutral;
  return { badgeColor: style.badgeColor, label };
}

export type RunsStatusFilter = "all" | "completed" | "errors" | "running" | "queued";

export function filterRunEvents(
  events: CanvasesCanvasEventWithExecutions[],
  nodes: ComponentsNode[],
  statusFilter: RunsStatusFilter,
  searchQuery: string,
) {
  const query = searchQuery.trim().toLowerCase();
  return events.filter((event) => {
    const executions = event.executions || [];
    if (statusFilter !== "all" && !matchesStatusFilter(executions, statusFilter)) return false;
    if (query && !matchesSearchQuery(event, executions, nodes, query)) return false;
    return true;
  });
}

function matchesStatusFilter(executions: CanvasesCanvasNodeExecutionRef[], statusFilter: RunsStatusFilter): boolean {
  const aggregate = executions.length > 0 ? getAggregateStatus(executions) : "queued";
  if (statusFilter === "completed") return aggregate === "completed" || aggregate === "cancelled";
  if (statusFilter === "errors") return aggregate === "error";
  if (statusFilter === "running") return aggregate === "running";
  if (statusFilter === "queued") return aggregate === "queued";
  return true;
}

function matchesSearchQuery(
  event: CanvasesCanvasEventWithExecutions,
  executions: CanvasesCanvasNodeExecutionRef[],
  nodes: ComponentsNode[],
  query: string,
): boolean {
  const triggerNode = nodes.find((n) => n.id === event.nodeId);
  const triggerRenderer = getTriggerRenderer(triggerNode?.trigger?.name || "");
  const eventInfo = buildEventInfo(event);
  const { title } = eventInfo ? triggerRenderer.getTitleAndSubtitle({ event: eventInfo }) : { title: "" };

  const searchableText = [
    event.id,
    title,
    triggerNode?.name,
    ...executions.map((e) => {
      const node = nodes.find((n) => n.id === e.nodeId);
      return [node?.name, node?.id, e.resultMessage].filter(Boolean).join(" ");
    }),
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();

  return searchableText.includes(query);
}

export function countUnacknowledgedErrors(events: CanvasesCanvasEventWithExecutions[]): number {
  let count = 0;
  for (const event of events) {
    for (const exec of event.executions || []) {
      if (exec.result === "RESULT_FAILED" && exec.resultReason !== "RESULT_REASON_ERROR_RESOLVED") {
        count++;
      }
    }
  }
  return count;
}

export function findNode(nodes: ComponentsNode[], nodeId: string | undefined): ComponentsNode | undefined {
  if (!nodeId) return undefined;
  const baseNodeId = nodeId.includes(":") ? nodeId.split(":")[0] : nodeId;
  return nodes.find((n) => n.id === nodeId) || nodes.find((n) => n.id === baseNodeId);
}

export function mergeQueueItemsWithEvents(
  events: CanvasesCanvasEventWithExecutions[],
  nodeQueueItemsMap: Record<string, CanvasesCanvasNodeQueueItem[]>,
): {
  queueItemsByEventId: Record<string, CanvasesCanvasNodeQueueItem[]>;
  allEvents: CanvasesCanvasEventWithExecutions[];
} {
  const map: Record<string, CanvasesCanvasNodeQueueItem[]> = {};
  const orphansByEvent: Record<
    string,
    { event: CanvasesCanvasNodeQueueItem["rootEvent"]; items: CanvasesCanvasNodeQueueItem[] }
  > = {};
  const eventIds = new Set(events.map((e) => e.id));

  for (const items of Object.values(nodeQueueItemsMap)) {
    for (const item of items) {
      const eventId = item.rootEvent?.id;
      if (!eventId) continue;
      if (eventIds.has(eventId)) {
        if (!map[eventId]) map[eventId] = [];
        map[eventId].push(item);
      } else {
        if (!orphansByEvent[eventId]) orphansByEvent[eventId] = { event: item.rootEvent, items: [] };
        orphansByEvent[eventId].items.push(item);
      }
    }
  }

  const orphanEvents: CanvasesCanvasEventWithExecutions[] = Object.entries(orphansByEvent).map(
    ([eventId, { event: rootEvent, items }]) => ({
      id: eventId,
      canvasId: items[0]?.canvasId,
      nodeId: rootEvent?.nodeId,
      channel: rootEvent?.channel,
      data: rootEvent?.data as Record<string, unknown> | undefined,
      createdAt: rootEvent?.createdAt,
      executions: [],
    }),
  );

  if (orphanEvents.length === 0) return { queueItemsByEventId: map, allEvents: events };

  for (const [eventId, { items }] of Object.entries(orphansByEvent)) {
    map[eventId] = items;
  }

  const merged = [...orphanEvents, ...events];
  merged.sort((a, b) => {
    const ta = a.createdAt ? new Date(a.createdAt).getTime() : 0;
    const tb = b.createdAt ? new Date(b.createdAt).getTime() : 0;
    return tb - ta;
  });
  return { queueItemsByEventId: map, allEvents: merged };
}

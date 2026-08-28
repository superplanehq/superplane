import type {
  FactoriesFactory,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { isActiveWorkOrderExecution } from "./workOrderExecutions";
import { formatUsdCents, formatWorkOrderUsage, parseWorkOrderMetric } from "./workOrderUsage";
import {
  WORK_ORDER_BOARD_LANES,
  WORK_ORDER_DISPLAY_STATUSES,
  getWorkOrderDisplayKey,
  getWorkOrderDisplayStatus,
  isUnassignedWorkOrder,
  type WorkOrderBoardLaneId,
  type WorkOrderDisplayStatus,
} from "./workOrderProgress";
import { visibleWorkOrdersForCollections } from "./workOrderVisibility";

/**
 * Presentation model built on top of a `FactoriesWorkOrder`. It centralizes
 * every derived value the Work Orders layouts need — identifiers, status,
 * latest line and step, usage, assignee summary, and search text — so
 * board/list/table stay display-only and can share the same reducers.
 *
 * The backend still owns the raw payload; this module is just a lens over
 * it. `workspace-identity-backend` filled in `key`/`number`; here we add
 * everything else the reference application shows.
 */
export interface WorkOrderListEntry {
  order: FactoriesWorkOrder;
  id: string;
  displayKey: string;
  /** Numeric part of `displayKey`, used to order by ID. */
  displayNumber: number;
  title: string;
  displayStatus: WorkOrderDisplayStatus;
  latestExecution: FactoriesWorkOrderExecution | null;
  latestLineName: string | null;
  latestStepName: string | null;
  /** Caption every layout shows under the title, empty when nothing ran yet. */
  lineStepLabel: string;
  lineIds: string[];
  lineNames: string[];
  distinctLineCount: number;
  totalTokens: number;
  totalCostCents: number;
  usageLabel: string | null;
  usageTooltip: string | null;
  updatedAtMs: number;
  createdAtMs: number;
  assigneeIds: string[];
  assigneeNames: string[];
  isUnassigned: boolean;
  searchHaystack: string;
  isDispatchable: boolean;
}

export function buildWorkOrderListEntry(
  order: FactoriesWorkOrder,
  factory: FactoriesFactory | null | undefined,
): WorkOrderListEntry {
  const dispatches = order.lineDispatches ?? [];
  const pairs = flattenDispatchExecutions(dispatches);
  const latestPair = findLatestExecution(pairs);
  const lines = collectLines(dispatches);
  const { totalTokens, totalCostCents } = sumUsage(
    order,
    pairs.map((pair) => pair.execution),
  );
  const { assigneeIds, assigneeNames } = collectAssignees(order);

  const createdAtMs = parseTimestamp(order.createdAt);
  const displayKey = getWorkOrderDisplayKey(order, factory?.key ?? null);
  const title = trimOrNull(order.title) ?? "Untitled work order";
  const latestLineName = trimOrNull(latestPair?.line?.name);
  const latestStepName = trimOrNull(latestPair?.execution.step);

  return {
    order,
    id: order.id ?? "",
    displayKey,
    displayNumber: parseIdentifierNumber(displayKey),
    title,
    displayStatus: getWorkOrderDisplayStatus(order),
    latestExecution: latestPair?.execution ?? null,
    latestLineName,
    latestStepName,
    lineStepLabel: buildLineStepLabel(latestLineName, latestStepName, lines.ids.length),
    lineIds: lines.ids,
    lineNames: lines.names,
    distinctLineCount: lines.ids.length,
    totalTokens,
    totalCostCents,
    usageLabel: formatWorkOrderUsage(totalTokens, totalCostCents),
    usageTooltip: formatUsageTooltip(totalTokens, totalCostCents),
    updatedAtMs: parseTimestamp(order.updatedAt) || createdAtMs,
    createdAtMs,
    assigneeIds,
    assigneeNames,
    isUnassigned: isUnassignedWorkOrder(order),
    searchHaystack: buildSearchHaystack([
      displayKey,
      title,
      order.description,
      lines.names.join(" "),
      latestStepName,
      assigneeNames.join(" "),
    ]),
    isDispatchable: isDispatchableState(order.state),
  };
}

function parseTimestamp(value: string | undefined): number {
  return Date.parse(value ?? "") || 0;
}

function trimOrNull(value: string | undefined): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

function isDispatchableState(state: FactoriesWorkOrder["state"]): boolean {
  return state === "STATE_DRAFT" || state === "STATE_OPEN";
}

function buildSearchHaystack(parts: Array<string | null | undefined>): string {
  return parts.filter(Boolean).join(" ").toLowerCase();
}

/**
 * "line · step" for the newest execution, plus a count of the other lines
 * the order touched. Built here so Board, List, and Table cannot drift
 * apart on the wording.
 */
function buildLineStepLabel(lineName: string | null, stepName: string | null, distinctLineCount: number): string {
  const label = [lineName, stepName].filter(Boolean).join(" · ");
  if (!label) {
    return "";
  }
  const otherLines = distinctLineCount - 1;
  if (otherLines < 1) {
    return label;
  }
  return `${label} · +${otherLines} more line${otherLines === 1 ? "" : "s"}`;
}

/** One step execution paired with the line ref of its parent dispatch —
 * `FactoriesWorkOrderExecution` no longer carries its own `line`, that
 * moved to the parent `FactoriesWorkOrderLineDispatch`. */
interface ExecutionWithLine {
  execution: FactoriesWorkOrderExecution;
  line?: FactoriesLineRef;
}

function flattenDispatchExecutions(dispatches: FactoriesWorkOrderLineDispatch[]): ExecutionWithLine[] {
  const pairs: ExecutionWithLine[] = [];
  for (const dispatch of dispatches) {
    for (const execution of dispatch.stepExecutions ?? []) {
      pairs.push({ execution, line: dispatch.line });
    }
  }
  return pairs;
}

/** Distinct lines the order has run on (one or more dispatches), in first-seen order. */
function collectLines(dispatches: FactoriesWorkOrderLineDispatch[]): { ids: string[]; names: string[] } {
  const ids = new Set<string>();
  const names = new Set<string>();
  for (const dispatch of dispatches) {
    const id = dispatch.line?.id?.trim();
    if (id) {
      ids.add(id);
    }
    const name = dispatch.line?.name?.trim();
    if (name) {
      names.add(name);
    }
  }
  return { ids: [...ids], names: [...names] };
}

/**
 * Order-level totals win when the backend has them; otherwise we add up
 * the executions so older payloads still show usage.
 */
function sumUsage(
  order: FactoriesWorkOrder,
  executions: FactoriesWorkOrderExecution[],
): { totalTokens: number; totalCostCents: number } {
  const totalTokens = parseWorkOrderMetric(order.totalTokens);
  const totalCostCents = parseWorkOrderMetric(order.totalCostCents);
  if (totalTokens > 0 || totalCostCents > 0) {
    return { totalTokens, totalCostCents };
  }
  return executions.reduce(
    (sum, execution) => ({
      totalTokens: sum.totalTokens + parseWorkOrderMetric(execution.totalTokens),
      totalCostCents: sum.totalCostCents + parseWorkOrderMetric(execution.costCents),
    }),
    { totalTokens: 0, totalCostCents: 0 },
  );
}

function collectAssignees(order: FactoriesWorkOrder): { assigneeIds: string[]; assigneeNames: string[] } {
  const owner = order.assignees?.[0];
  if (!owner?.id) {
    return { assigneeIds: [], assigneeNames: [] };
  }
  const name = (owner.name ?? "").trim();
  return {
    assigneeIds: [owner.id],
    assigneeNames: name ? [name] : [],
  };
}

export function buildWorkOrderListEntries(
  orders: FactoriesWorkOrder[],
  factory: FactoriesFactory | null | undefined,
): WorkOrderListEntry[] {
  return visibleWorkOrdersForCollections(orders).map((order) => buildWorkOrderListEntry(order, factory));
}

/**
 * Latest execution across every line dispatch, ordered by `updatedAt` then
 * `createdAt`. An active execution wins ties so an in-flight step surfaces
 * over a stale finished one that happens to share a timestamp. "Active"
 * must match the predicate behind the Running display status, or a tie can
 * show a line and step that disagree with the status pill.
 */
function findLatestExecution(pairs: ExecutionWithLine[]): ExecutionWithLine | null {
  if (pairs.length === 0) {
    return null;
  }
  let winner: ExecutionWithLine | null = null;
  let winnerAt = -Infinity;
  for (const pair of pairs) {
    const at = Date.parse(pair.execution.updatedAt ?? pair.execution.createdAt ?? "") || 0;
    if (at > winnerAt || (at === winnerAt && isActiveWorkOrderExecution(pair.execution))) {
      winner = pair;
      winnerAt = at;
    }
  }
  return winner;
}

function formatUsageTooltip(totalTokens: number, totalCostCents: number): string | null {
  const parts: string[] = [];
  if (totalCostCents > 0) {
    parts.push(formatUsdCents(totalCostCents));
  }
  if (totalTokens > 0) {
    parts.push(`${totalTokens.toLocaleString()} tokens`);
  }
  return parts.length > 0 ? parts.join(" · ") : null;
}

/**
 * Scope pills next to the page title. `active` (Needs attention) keeps
 * drafts, waiting work, and failed runs. `my` keeps work assigned to the
 * viewer.
 */
export type WorkOrderScope = "all" | "active" | "my";

export const WORK_ORDER_SCOPES: Array<{ id: WorkOrderScope; label: string; tooltip: string }> = [
  { id: "all", label: "All", tooltip: "Every work order in this workspace." },
  { id: "active", label: "Needs attention", tooltip: "Work orders that need your attention." },
  {
    id: "my",
    label: "My",
    tooltip: "Work orders you created or started. SuperPlane assigns those to you.",
  },
];

/** Statuses that the Needs attention scope keeps. Running is in flight. */
const ACTIVE_SCOPE_STATUSES: WorkOrderDisplayStatus[] = ["draft", "waiting", "failed"];

/** Ordering options in the Display menu. `updated` is the default. */
export type WorkOrderOrdering = "updated" | "status" | "spend" | "key";

export const WORK_ORDER_ORDERINGS: Array<{ id: WorkOrderOrdering; label: string }> = [
  { id: "updated", label: "Updated" },
  { id: "status", label: "Status" },
  { id: "spend", label: "Spend" },
  { id: "key", label: "ID" },
];

/** Layout for the main body. Persisted across sessions. */
export type WorkOrderLayoutId = "board" | "list" | "table";

export const WORK_ORDER_LAYOUTS: Array<{ id: WorkOrderLayoutId; label: string }> = [
  { id: "board", label: "Board" },
  { id: "list", label: "List" },
  { id: "table", label: "Table" },
];

/** Sentinel assignee filter value that matches work orders with no assignee. */
export const UNASSIGNED_FILTER_VALUE = "unassigned";

/**
 * Filters chosen in the Filter menu. Each dimension narrows independently
 * (AND across dimensions, OR within one), and an empty array means the
 * dimension is not filtering at all.
 */
export interface WorkOrderFilters {
  statuses: WorkOrderDisplayStatus[];
  lineIds: string[];
  assigneeIds: string[];
}

export const EMPTY_WORK_ORDER_FILTERS: WorkOrderFilters = { statuses: [], lineIds: [], assigneeIds: [] };

export function countWorkOrderFilters(filters: WorkOrderFilters): number {
  return filters.statuses.length + filters.lineIds.length + filters.assigneeIds.length;
}

export function applyWorkOrderScope(
  entries: WorkOrderListEntry[],
  scope: WorkOrderScope,
  currentUserId?: string,
): WorkOrderListEntry[] {
  if (scope === "all") {
    return entries;
  }
  if (scope === "active") {
    return entries.filter((entry) => ACTIVE_SCOPE_STATUSES.includes(entry.displayStatus));
  }
  if (!currentUserId) {
    return [];
  }
  return entries.filter((entry) => entry.assigneeIds.includes(currentUserId));
}

export function applyWorkOrderFilters(entries: WorkOrderListEntry[], filters: WorkOrderFilters): WorkOrderListEntry[] {
  let result = entries;
  if (filters.statuses.length > 0) {
    result = result.filter((entry) => filters.statuses.includes(entry.displayStatus));
  }
  if (filters.lineIds.length > 0) {
    result = result.filter((entry) => entry.lineIds.some((lineId) => filters.lineIds.includes(lineId)));
  }
  if (filters.assigneeIds.length > 0) {
    result = result.filter((entry) => {
      if (entry.assigneeIds.some((id) => filters.assigneeIds.includes(id))) {
        return true;
      }
      return filters.assigneeIds.includes(UNASSIGNED_FILTER_VALUE) && entry.assigneeIds.length === 0;
    });
  }
  return result;
}

export function applyWorkOrderSearch(entries: WorkOrderListEntry[], query: string): WorkOrderListEntry[] {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return entries;
  }
  return entries.filter((entry) => entry.searchHaystack.includes(needle));
}

export function applyWorkOrderOrdering(
  entries: WorkOrderListEntry[],
  ordering: WorkOrderOrdering,
): WorkOrderListEntry[] {
  const sorted = [...entries];
  switch (ordering) {
    case "updated":
      sorted.sort((a, b) => b.updatedAtMs - a.updatedAtMs || a.title.localeCompare(b.title));
      break;
    case "status":
      sorted.sort(
        (a, b) =>
          WORK_ORDER_DISPLAY_STATUSES.indexOf(a.displayStatus) - WORK_ORDER_DISPLAY_STATUSES.indexOf(b.displayStatus) ||
          b.updatedAtMs - a.updatedAtMs,
      );
      break;
    case "spend":
      sorted.sort((a, b) => b.totalCostCents - a.totalCostCents || b.totalTokens - a.totalTokens);
      break;
    case "key":
      sorted.sort((a, b) => b.displayNumber - a.displayNumber || a.displayKey.localeCompare(b.displayKey));
      break;
  }
  return sorted;
}

/**
 * Buckets entries into the shared board lanes, keeping the incoming order
 * inside each lane. Board and List both group this way so a work order
 * never lands in different sections depending on the layout.
 */
export function groupWorkOrderEntriesByLane(
  entries: WorkOrderListEntry[],
): Map<WorkOrderBoardLaneId, WorkOrderListEntry[]> {
  const grouped = new Map<WorkOrderBoardLaneId, WorkOrderListEntry[]>(
    WORK_ORDER_BOARD_LANES.map((lane) => [lane.id, []]),
  );
  for (const entry of entries) {
    const lane = WORK_ORDER_BOARD_LANES.find((candidate) => candidate.statuses.includes(entry.displayStatus));
    if (lane) {
      grouped.get(lane.id)?.push(entry);
    }
  }
  return grouped;
}

/** Numeric suffix of `SP-42`, so `SP-10` sorts above `SP-9`. */
function parseIdentifierNumber(displayKey: string): number {
  const match = /-(\d+)$/.exec(displayKey);
  return match ? Number(match[1]) : 0;
}

/**
 * Storybook spending explorer: filter and roll up a ledger that matches
 * `workspace_usage_events` (model tokens + runner VM time).
 *
 * User grouping is not a column on the ledger today. The explorer attributes
 * spend to the task owner (work-order `created_by`) so the filter can stay
 * honest to how SuperPlane already stores usage.
 */

import { formatCompactTokens, formatDurationSeconds, formatUsdCents } from "../../../lib/workOrderUsage";

export type SpendingUsageKind = "model" | "compute";
export type SpendingFundingSource = "hosted" | "byok";
export type SpendingPeriodPreset = "day" | "week" | "month" | "year" | "custom";
export type SpendingBreakdown = "workspace" | "user" | "model" | "machine";

export interface SpendingCatalogItem {
  id: string;
  label: string;
}

export interface SpendingUsageEvent {
  id: string;
  occurredAt: string;
  factoryId: string;
  userId: string;
  provider: string;
  model: string;
  usageKind: SpendingUsageKind;
  fundingSource: SpendingFundingSource;
  machineType: string;
  totalTokens: number;
  durationSeconds: number;
  costCents: number;
}

export interface SpendingDateRange {
  /** Inclusive. */
  start: Date;
  /** Exclusive. */
  end: Date;
}

export interface SpendingFilters {
  userId: string;
  workspaceId: string;
  model: string;
  machineType: string;
}

export interface SpendingTotals {
  costCents: number;
  tokens: number;
  durationSeconds: number;
  hostedCostCents: number;
  byokCostCents: number;
}

export interface SpendingSeriesPoint {
  key: string;
  label: string;
  totalCents: number;
  values: Record<string, number>;
}

export interface SpendingBreakdownRow {
  id: string;
  label: string;
  tokens: number;
  durationSeconds: number;
  costCents: number;
  share: number;
}

export interface SpendingReport {
  range: SpendingDateRange;
  totals: SpendingTotals;
  series: SpendingSeriesPoint[];
  seriesKeys: SpendingCatalogItem[];
  breakdown: SpendingBreakdownRow[];
}

export const EMPTY_SPENDING_FILTERS: SpendingFilters = {
  userId: "",
  workspaceId: "",
  model: "",
  machineType: "",
};

export const SPENDING_PERIOD_PRESETS: Array<{
  value: Exclude<SpendingPeriodPreset, "custom">;
  label: string;
}> = [
  { value: "day", label: "Last 24 hours" },
  { value: "week", label: "Last 7 days" },
  { value: "month", label: "Last 30 days" },
  { value: "year", label: "Last 12 months" },
];

export const SPENDING_BREAKDOWN_OPTIONS: Array<{ value: SpendingBreakdown; label: string }> = [
  { value: "workspace", label: "Workspaces" },
  { value: "user", label: "Users" },
  { value: "model", label: "Models" },
  { value: "machine", label: "Machine types" },
];

export const MODEL_BREAKDOWN_OPTIONS = SPENDING_BREAKDOWN_OPTIONS.filter((option) => option.value !== "machine");
export const MACHINE_BREAKDOWN_OPTIONS = SPENDING_BREAKDOWN_OPTIONS.filter((option) => option.value !== "model");

const DAY_MS = 24 * 60 * 60 * 1000;
const HOUR_MS = 60 * 60 * 1000;
const OTHER_SERIES_ID = "other";

export function modelKey(provider: string, model: string): string {
  return `${provider}/${model}`;
}

export function rangeForPreset(preset: Exclude<SpendingPeriodPreset, "custom">, now: Date): SpendingDateRange {
  const end = new Date(now.getTime());
  if (preset === "day") {
    return { start: new Date(now.getTime() - DAY_MS), end };
  }
  if (preset === "week") {
    return { start: new Date(now.getTime() - 7 * DAY_MS), end };
  }
  if (preset === "month") {
    return { start: new Date(now.getTime() - 30 * DAY_MS), end };
  }
  return { start: new Date(now.getTime() - 365 * DAY_MS), end };
}

/**
 * Bucket size used to stabilize the "now" anchor for preset ranges.
 *
 * Rounding "now" down to the start of the current minute means quick
 * remounts (switching settings tabs and back) resolve to the exact same
 * range, so the spending report query cache is hit instead of starting a
 * brand-new query on every mount. A stale-but-fresh-enough end time is a
 * fine trade-off: `useOrganizationSpendingReport`'s `staleTime` still
 * triggers a background refetch to catch up.
 */
const SPENDING_NOW_QUANTIZE_MS = 60 * 1000;

export function quantizeSpendingNow(now: Date): Date {
  const quantized = Math.floor(now.getTime() / SPENDING_NOW_QUANTIZE_MS) * SPENDING_NOW_QUANTIZE_MS;
  return new Date(quantized);
}

export function rangeFromCustomDays(from: Date, to: Date): SpendingDateRange {
  const start = startOfUtcDay(from);
  const endDay = startOfUtcDay(to);
  return { start, end: new Date(endDay.getTime() + DAY_MS) };
}

export function hasActiveSpendingFilters(filters: SpendingFilters): boolean {
  return Boolean(filters.userId || filters.workspaceId || filters.model || filters.machineType);
}

export function filterSpendingEvents(
  events: SpendingUsageEvent[],
  range: SpendingDateRange,
  filters: SpendingFilters,
  usageKind?: SpendingUsageKind,
): SpendingUsageEvent[] {
  return events.filter((event) => eventMatches(event, range, filters, usageKind));
}

export function buildSpendingReport({
  events,
  range,
  filters,
  breakdown,
  catalogs,
  usageKind,
}: {
  events: SpendingUsageEvent[];
  range: SpendingDateRange;
  filters: SpendingFilters;
  breakdown: SpendingBreakdown;
  catalogs: SpendingCatalogs;
  usageKind: SpendingUsageKind;
}): SpendingReport {
  const matched = filterSpendingEvents(events, range, filters, usageKind);
  const totals = sumSpendingTotals(matched);
  const grain = chartGrainForRange(range);
  const grouped = groupBreakdown(matched, breakdown, catalogs);
  const seriesKeys = seriesKeysForBreakdown(grouped, breakdown);
  const series = bucketEvents(matched, range, grain, breakdown, seriesKeys);

  return {
    range,
    totals,
    series,
    seriesKeys,
    breakdown: grouped,
  };
}

export interface SpendingCatalogs {
  users: SpendingCatalogItem[];
  workspaces: SpendingCatalogItem[];
  models: SpendingCatalogItem[];
  machines: SpendingCatalogItem[];
}

export function sumSpendingTotals(events: SpendingUsageEvent[]): SpendingTotals {
  return events.reduce<SpendingTotals>(
    (totals, event) => {
      totals.costCents += event.costCents;
      totals.tokens += event.totalTokens;
      totals.durationSeconds += event.durationSeconds;
      if (event.fundingSource === "hosted") {
        totals.hostedCostCents += event.costCents;
      } else {
        totals.byokCostCents += event.costCents;
      }
      return totals;
    },
    { costCents: 0, tokens: 0, durationSeconds: 0, hostedCostCents: 0, byokCostCents: 0 },
  );
}

export function formatSpendingRangeCaption(range: SpendingDateRange): string {
  const displayEnd = new Date(range.end.getTime() - 1);
  const startLabel = formatUtcDay(range.start);
  const endLabel = formatUtcDay(displayEnd);
  if (startLabel === endLabel) {
    return startLabel;
  }
  return `${startLabel} – ${endLabel}`;
}

export function spendingPeriodTriggerLabel(period: SpendingPeriodPreset, range: SpendingDateRange): string {
  const preset = SPENDING_PERIOD_PRESETS.find((option) => option.value === period);
  if (preset) {
    return preset.label;
  }
  return formatSpendingRangeCaption(range);
}

export function formatFilterTriggerLabel(allLabel: string, selectedLabel?: string): string {
  return selectedLabel || allLabel;
}

export function formatShare(share: number): string {
  if (share <= 0) {
    return "0%";
  }
  return `${Math.round(share * 100)}%`;
}

export function spendingMetricCopy(totals: SpendingTotals): {
  spend: string;
  tokens: string;
  duration: string;
  hosted: string;
  byok: string;
} {
  return {
    spend: formatUsdCents(totals.costCents),
    tokens: formatCompactTokens(totals.tokens),
    duration: formatDurationSeconds(totals.durationSeconds),
    hosted: formatUsdCents(totals.hostedCostCents),
    byok: formatUsdCents(totals.byokCostCents),
  };
}

type ChartGrain = "hour" | "day" | "month";

export function spendingTimeGrainForRange(range: SpendingDateRange): ChartGrain {
  return chartGrainForRange(range);
}

function eventMatches(
  event: SpendingUsageEvent,
  range: SpendingDateRange,
  filters: SpendingFilters,
  usageKind?: SpendingUsageKind,
): boolean {
  const occurred = Date.parse(event.occurredAt);
  if (Number.isNaN(occurred) || occurred < range.start.getTime() || occurred >= range.end.getTime()) {
    return false;
  }
  if (usageKind && event.usageKind !== usageKind) {
    return false;
  }
  if (filters.userId && event.userId !== filters.userId) {
    return false;
  }
  if (filters.workspaceId && event.factoryId !== filters.workspaceId) {
    return false;
  }
  if (filters.model && modelKey(event.provider, event.model) !== filters.model) {
    return false;
  }
  if (filters.machineType && event.machineType !== filters.machineType) {
    return false;
  }
  return true;
}

function chartGrainForRange(range: SpendingDateRange): ChartGrain {
  const span = range.end.getTime() - range.start.getTime();
  if (span <= 2 * DAY_MS) {
    return "hour";
  }
  if (span <= 90 * DAY_MS) {
    return "day";
  }
  return "month";
}

function groupBreakdown(
  events: SpendingUsageEvent[],
  breakdown: SpendingBreakdown,
  catalogs: SpendingCatalogs,
): SpendingBreakdownRow[] {
  const totalsById = new Map<string, { tokens: number; durationSeconds: number; costCents: number }>();
  for (const event of events) {
    if (breakdown === "model" && event.usageKind !== "model") {
      continue;
    }
    if (breakdown === "machine" && event.usageKind !== "compute") {
      continue;
    }
    const id = breakdownId(event, breakdown);
    const current = totalsById.get(id) ?? { tokens: 0, durationSeconds: 0, costCents: 0 };
    current.tokens += event.totalTokens;
    current.durationSeconds += event.durationSeconds;
    current.costCents += event.costCents;
    totalsById.set(id, current);
  }

  const totalCost = [...totalsById.values()].reduce((sum, row) => sum + row.costCents, 0);
  const labels = labelCatalog(breakdown, catalogs);

  return [...totalsById.entries()]
    .map(([id, row]) => ({
      id,
      label: labels.get(id) ?? id,
      tokens: row.tokens,
      durationSeconds: row.durationSeconds,
      costCents: row.costCents,
      share: totalCost > 0 ? row.costCents / totalCost : 0,
    }))
    .sort((left, right) => right.costCents - left.costCents || left.label.localeCompare(right.label));
}

function seriesKeysForBreakdown(rows: SpendingBreakdownRow[], breakdown: SpendingBreakdown): SpendingCatalogItem[] {
  const top = rows.slice(0, 5).map((row) => ({ id: row.id, label: row.label }));
  if (rows.length > 5) {
    top.push({ id: OTHER_SERIES_ID, label: otherSeriesLabel(breakdown) });
  }
  return top;
}

function bucketEvents(
  events: SpendingUsageEvent[],
  range: SpendingDateRange,
  grain: ChartGrain,
  breakdown: SpendingBreakdown,
  seriesKeys: SpendingCatalogItem[],
): SpendingSeriesPoint[] {
  const buckets = emptyBuckets(range, grain);
  const knownIds = new Set(seriesKeys.map((item) => item.id));
  const hasOther = knownIds.has(OTHER_SERIES_ID);

  for (const event of events) {
    const bucketKey = bucketKeyFor(event.occurredAt, grain);
    const point = buckets.get(bucketKey);
    if (!point) {
      continue;
    }
    const id = seriesIdForEvent(event, breakdown, knownIds, hasOther);
    if (!id) {
      continue;
    }
    point.values[id] = (point.values[id] ?? 0) + event.costCents;
    point.totalCents += event.costCents;
  }

  return [...buckets.values()];
}

function seriesIdForEvent(
  event: SpendingUsageEvent,
  breakdown: SpendingBreakdown,
  knownIds: Set<string>,
  hasOther: boolean,
): string | undefined {
  if (breakdown === "model" && event.usageKind !== "model") {
    return undefined;
  }
  if (breakdown === "machine" && event.usageKind !== "compute") {
    return undefined;
  }
  const id = breakdownId(event, breakdown);
  if (knownIds.has(id)) {
    return id;
  }
  return hasOther ? OTHER_SERIES_ID : undefined;
}

function emptyBuckets(range: SpendingDateRange, grain: ChartGrain): Map<string, SpendingSeriesPoint> {
  const buckets = new Map<string, SpendingSeriesPoint>();
  if (grain === "hour") {
    for (let time = range.start.getTime(); time < range.end.getTime(); time += HOUR_MS) {
      const at = new Date(time);
      const key = hourKey(at);
      buckets.set(key, { key, label: formatHourLabel(at), totalCents: 0, values: {} });
    }
    return buckets;
  }
  if (grain === "day") {
    for (let time = startOfUtcDay(range.start).getTime(); time < range.end.getTime(); time += DAY_MS) {
      const at = new Date(time);
      const key = dayKey(at);
      buckets.set(key, { key, label: formatDayTick(at), totalCents: 0, values: {} });
    }
    return buckets;
  }
  let cursor = new Date(Date.UTC(range.start.getUTCFullYear(), range.start.getUTCMonth(), 1));
  while (cursor.getTime() < range.end.getTime()) {
    const key = monthKey(cursor);
    buckets.set(key, { key, label: formatMonthTick(cursor), totalCents: 0, values: {} });
    cursor = new Date(Date.UTC(cursor.getUTCFullYear(), cursor.getUTCMonth() + 1, 1));
  }
  return buckets;
}

function bucketKeyFor(occurredAt: string, grain: ChartGrain): string {
  const at = new Date(occurredAt);
  if (grain === "hour") {
    return hourKey(at);
  }
  if (grain === "day") {
    return dayKey(at);
  }
  return monthKey(at);
}

function breakdownId(event: SpendingUsageEvent, breakdown: SpendingBreakdown): string {
  if (breakdown === "workspace") {
    return event.factoryId;
  }
  if (breakdown === "user") {
    return event.userId;
  }
  if (breakdown === "model") {
    return modelKey(event.provider, event.model);
  }
  return event.machineType;
}

function labelCatalog(breakdown: SpendingBreakdown, catalogs: SpendingCatalogs): Map<string, string> {
  const items =
    breakdown === "workspace"
      ? catalogs.workspaces
      : breakdown === "user"
        ? catalogs.users
        : breakdown === "model"
          ? catalogs.models
          : catalogs.machines;
  return new Map(items.map((item) => [item.id, item.label]));
}

function otherSeriesLabel(breakdown: SpendingBreakdown): string {
  if (breakdown === "workspace") {
    return "Other workspaces";
  }
  if (breakdown === "user") {
    return "Other users";
  }
  if (breakdown === "model") {
    return "Other models";
  }
  return "Other machines";
}

function startOfUtcDay(value: Date): Date {
  return new Date(Date.UTC(value.getUTCFullYear(), value.getUTCMonth(), value.getUTCDate()));
}

function hourKey(value: Date): string {
  return `${dayKey(value)}T${String(value.getUTCHours()).padStart(2, "0")}`;
}

function dayKey(value: Date): string {
  return `${value.getUTCFullYear()}-${String(value.getUTCMonth() + 1).padStart(2, "0")}-${String(value.getUTCDate()).padStart(2, "0")}`;
}

function monthKey(value: Date): string {
  return `${value.getUTCFullYear()}-${String(value.getUTCMonth() + 1).padStart(2, "0")}`;
}

function formatUtcDay(value: Date): string {
  return value.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });
}

function formatHourLabel(value: Date): string {
  return `${String(value.getUTCHours()).padStart(2, "0")}:00`;
}

function formatDayTick(value: Date): string {
  return value.toLocaleDateString("en-US", { month: "short", day: "numeric", timeZone: "UTC" });
}

function formatMonthTick(value: Date): string {
  return value.toLocaleDateString("en-US", { month: "short", year: "2-digit", timeZone: "UTC" });
}

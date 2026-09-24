import {
  DAY_MS,
  HOUR_MS,
  dayKey,
  formatDayTick,
  formatHourLabel,
  formatMonthTick,
  hourKey,
  monthKey,
  startOfUtcDay,
} from "./spendingRedesignTime";
import {
  modelKey,
  SPENDING_FUNDING_SOURCE_OPTIONS,
  type SpendingBreakdown,
  type SpendingBreakdownRow,
  type SpendingCatalogItem,
  type SpendingCatalogs,
  type SpendingDateRange,
  type SpendingFilters,
  type SpendingReport,
  type SpendingSeriesPoint,
  type SpendingTotals,
  type SpendingUsageEvent,
  type SpendingUsageKind,
} from "./spendingRedesignLib";

const OTHER_SERIES_ID = "other";

type ChartGrain = "hour" | "day" | "month";

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

/**
 * Drop series that do not match the selected group-by filter.
 *
 * Production keeps the previous API report on screen while a new filter
 * fetches. Narrow only when the report already has that dimension, so a
 * source filter does not wipe a workspace-grouped chart.
 */
export function narrowSpendingReport(
  report: SpendingReport,
  filters: SpendingFilters,
  breakdown: SpendingBreakdown,
): SpendingReport {
  const selected = selectedBreakdownFilter(filters, breakdown);
  if (!selected) {
    return report;
  }
  if (breakdown === "funding_source" && !reportHasFundingSourceIds(report)) {
    return report;
  }

  const seriesKeys = report.seriesKeys.filter((item) => item.id === selected);
  const breakdownRows = report.breakdown.filter((row) => row.id === selected);
  const series = report.series.map((point) => {
    const value = point.values[selected] ?? 0;
    return {
      ...point,
      totalCents: value,
      values: seriesKeys.length > 0 ? { [selected]: value } : {},
    };
  });
  const costCents = breakdownRows.reduce((sum, row) => sum + row.costCents, 0);
  const tokens = breakdownRows.reduce((sum, row) => sum + row.tokens, 0);
  const durationSeconds = breakdownRows.reduce((sum, row) => sum + row.durationSeconds, 0);

  return {
    ...report,
    series,
    seriesKeys,
    breakdown: breakdownRows.map((row) => ({ ...row, share: costCents > 0 ? row.costCents / costCents : 0 })),
    totals: {
      costCents,
      tokens,
      durationSeconds,
      hostedCostCents: breakdown === "funding_source" && selected === "hosted" ? costCents : 0,
      byokCostCents: breakdown === "funding_source" && selected === "byok" ? costCents : 0,
    },
  };
}

function reportHasFundingSourceIds(report: SpendingReport): boolean {
  return (
    report.seriesKeys.some((item) => item.id === "hosted" || item.id === "byok") ||
    report.breakdown.some((row) => row.id === "hosted" || row.id === "byok")
  );
}

function selectedBreakdownFilter(filters: SpendingFilters, breakdown: SpendingBreakdown): string {
  if (breakdown === "workspace") {
    return filters.workspaceId;
  }
  if (breakdown === "user") {
    return filters.userId;
  }
  if (breakdown === "model") {
    return filters.model;
  }
  if (breakdown === "machine") {
    return filters.machineType;
  }
  return filters.fundingSource;
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

export function spendingTimeGrainForRange(range: SpendingDateRange): ChartGrain {
  return chartGrainForRange(range);
}

function eventMatches(
  event: SpendingUsageEvent,
  range: SpendingDateRange,
  filters: SpendingFilters,
  usageKind?: SpendingUsageKind,
): boolean {
  if (!spendingEventInRange(event, range)) {
    return false;
  }
  if (usageKind && event.usageKind !== usageKind) {
    return false;
  }
  return spendingEventMatchesFilters(event, filters);
}

function spendingEventInRange(event: SpendingUsageEvent, range: SpendingDateRange): boolean {
  const occurred = Date.parse(event.occurredAt);
  return !Number.isNaN(occurred) && occurred >= range.start.getTime() && occurred < range.end.getTime();
}

function spendingEventMatchesFilters(event: SpendingUsageEvent, filters: SpendingFilters): boolean {
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
  if (filters.fundingSource && event.fundingSource !== filters.fundingSource) {
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
  if (breakdown === "funding_source") {
    return event.fundingSource;
  }
  return event.machineType;
}

function labelCatalog(breakdown: SpendingBreakdown, catalogs: SpendingCatalogs): Map<string, string> {
  if (breakdown === "funding_source") {
    return new Map(SPENDING_FUNDING_SOURCE_OPTIONS.map((item) => [item.id, item.label]));
  }
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
  if (breakdown === "funding_source") {
    return "Other sources";
  }
  return "Other machines";
}

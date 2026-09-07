/**
 * Storybook spending explorer: filter and roll up a ledger that matches
 * `workspace_usage_events` (model tokens + runner VM time).
 *
 * User grouping is not a column on the ledger today. The explorer attributes
 * spend to the task owner (work-order `created_by`) so the filter can stay
 * honest to how SuperPlane already stores usage.
 */

import { formatCompactTokens, formatDurationSeconds, formatUsdCents } from "../../../lib/workOrderUsage";
import { DAY_MS, formatUtcDay, startOfUtcDay } from "./spendingRedesignTime";

export type SpendingUsageKind = "model" | "compute";
export type SpendingFundingSource = "hosted" | "byok";
export type SpendingPeriodPreset = "day" | "week" | "month" | "year" | "custom";
export type SpendingBreakdown = "workspace" | "user" | "model" | "machine" | "funding_source";

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
  fundingSource: string;
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
  fundingSource: "",
};

export const SPENDING_FUNDING_SOURCE_OPTIONS: SpendingCatalogItem[] = [
  { id: "hosted", label: "SuperPlane-hosted" },
  { id: "byok", label: "Your keys" },
];

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
  { value: "funding_source", label: "Source" },
  { value: "workspace", label: "Workspaces" },
  { value: "user", label: "Users" },
  { value: "model", label: "Models" },
  { value: "machine", label: "Machine types" },
];

export const MODEL_BREAKDOWN_OPTIONS = SPENDING_BREAKDOWN_OPTIONS.filter((option) => option.value !== "machine");
export const MACHINE_BREAKDOWN_OPTIONS = SPENDING_BREAKDOWN_OPTIONS.filter(
  (option) => option.value !== "model" && option.value !== "funding_source",
);

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
  return Boolean(
    filters.userId || filters.workspaceId || filters.model || filters.machineType || filters.fundingSource,
  );
}

export interface SpendingCatalogs {
  users: SpendingCatalogItem[];
  workspaces: SpendingCatalogItem[];
  models: SpendingCatalogItem[];
  machines: SpendingCatalogItem[];
}

export {
  buildSpendingReport,
  filterSpendingEvents,
  spendingTimeGrainForRange,
  sumSpendingTotals,
} from "./spendingRedesignReport";

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

export function spendingBreakdownLabel(
  breakdown: SpendingBreakdown,
  options: Array<{ value: SpendingBreakdown; label: string }>,
): string {
  return options.find((option) => option.value === breakdown)?.label ?? "Workspaces";
}

export function spendingBreakdownColumnLabel(breakdown: SpendingBreakdown): string {
  if (breakdown === "workspace") {
    return "Workspace";
  }
  if (breakdown === "user") {
    return "User";
  }
  if (breakdown === "model") {
    return "Model";
  }
  if (breakdown === "funding_source") {
    return "Source";
  }
  return "Machine type";
}

export function spendingUsageCopy(kind: SpendingUsageKind): {
  title: string;
  description: string;
  emptyMessage: string;
  testIdPrefix: string;
  breakdownOptions: Array<{ value: SpendingBreakdown; label: string }>;
} {
  if (kind === "model") {
    return {
      title: "Model usage",
      description: "SuperPlane-hosted model usage uses hosted credit. Your keys usage is estimated and is not billed.",
      emptyMessage: "No model usage is recorded for this period.",
      testIdPrefix: "spending-model",
      breakdownOptions: MODEL_BREAKDOWN_OPTIONS,
    };
  }
  return {
    title: "VM usage",
    description: "Estimated spend in dollars for SuperPlane runner machines.",
    emptyMessage: "No VM usage is recorded for this period.",
    testIdPrefix: "spending-vm",
    breakdownOptions: MACHINE_BREAKDOWN_OPTIONS,
  };
}

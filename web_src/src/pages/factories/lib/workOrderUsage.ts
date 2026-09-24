import { formatCompactTokenLabel, formatCompactTokenValue } from "@/lib/formatTokenCount";
import { machineTypeLabel } from "@/lib/machineType";

export function parseWorkOrderMetric(value: string | number | undefined): number {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed : 0;
}

/** First metric that is greater than zero. Gateway emit-unpopulated sends `"0"`. */
export function firstPositiveWorkOrderMetric(...values: Array<string | number | undefined>): string | undefined {
  for (const value of values) {
    if (parseWorkOrderMetric(value) > 0) {
      return typeof value === "number" ? String(value) : value;
    }
  }
  return undefined;
}

export function formatCompactTokens(tokens: number): string {
  return formatCompactTokenLabel(tokens);
}

export function formatUsageTokenCount(tokens: number): string {
  return tokens > 0 ? formatCompactTokenValue(tokens) : "—";
}

export function formatUsageDuration(seconds: number): string {
  return seconds > 0 ? formatDurationSeconds(seconds) : "—";
}

const MICROS_PER_DOLLAR = 1_000_000;
const MICROS_PER_CENT = 10_000;

/** Dollar amount for a usage column. Zero spend is an em dash. */
export function formatUsageSpend(cents: number): string {
  return cents > 0 ? formatUsdCents(cents) : "—";
}

/** Dollar amount from ledger micros. Sub-cent spend stays visible. */
export function formatUsageSpendMicros(micros: number): string {
  return micros > 0 ? formatUsdMicros(micros) : "—";
}

/** Hosted model spend plus your-keys model spend. */
export function usageTokenSpendCents(hostedCostCents: number, byokCostCents: number): number {
  return hostedCostCents + byokCostCents;
}

export function usageTokenSpendMicros(hostedCostMicros: number, byokCostMicros: number): number {
  return hostedCostMicros + byokCostMicros;
}

/** VM spend is the remainder after model spend. */
export function usageVmSpendCents(totalCostCents: number, hostedCostCents: number, byokCostCents: number): number {
  return Math.max(0, totalCostCents - hostedCostCents - byokCostCents);
}

export function usageVmSpendMicros(totalCostMicros: number, hostedCostMicros: number, byokCostMicros: number): number {
  return Math.max(0, totalCostMicros - hostedCostMicros - byokCostMicros);
}

/**
 * Prefers ledger micros. Rows that predate those fields still format from
 * whole cents, including MSW fixtures that omit micros.
 */
export function usageSpendMicros(micros: string | number | undefined, cents: string | number | undefined): number {
  if (micros !== undefined && micros !== "") {
    return parseWorkOrderMetric(micros);
  }
  return parseWorkOrderMetric(cents) * MICROS_PER_CENT;
}

export function formatUsdCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

export type WorkOrderUsageByModel = {
  provider?: string;
  model?: string;
  totalTokens?: string | number;
  costCents?: string | number;
};

export type WorkOrderUsageByMachineType = {
  machineType?: string;
  durationSeconds?: string | number;
  costCents?: string | number;
};

export type WorkOrderSpendBreakdownRow = {
  label: string;
  detail: string;
  spend: string;
};

function usageModelDisplayName(provider?: string, model?: string): string {
  const raw = (model ?? "").trim() || (provider ?? "").trim();
  const slash = raw.lastIndexOf("/");
  if (slash >= 0 && slash < raw.length - 1) {
    return raw.slice(slash + 1);
  }
  return raw;
}

/** Model and machine-time rows that have spend. Empty when there is nothing to show. */
export function workOrderSpendBreakdownRows(
  byModel: WorkOrderUsageByModel[] | undefined,
  byMachine: WorkOrderUsageByMachineType[] | undefined,
): WorkOrderSpendBreakdownRow[] {
  const rows: WorkOrderSpendBreakdownRow[] = [];
  for (const row of byModel ?? []) {
    const cents = parseWorkOrderMetric(row.costCents);
    if (cents <= 0) {
      continue;
    }
    const label = usageModelDisplayName(row.provider, row.model);
    if (!label) {
      continue;
    }
    const tokens = parseWorkOrderMetric(row.totalTokens);
    rows.push({
      label,
      detail: tokens > 0 ? formatCompactTokens(tokens) : "",
      spend: formatUsdCents(cents),
    });
  }

  let machineCents = 0;
  let machineSeconds = 0;
  for (const row of byMachine ?? []) {
    machineCents += parseWorkOrderMetric(row.costCents);
    machineSeconds += parseWorkOrderMetric(row.durationSeconds);
  }
  if (machineCents > 0) {
    rows.push({
      label: "Machine time",
      detail: machineSeconds > 0 ? formatDurationSeconds(machineSeconds) : "",
      spend: formatUsdCents(machineCents),
    });
  }
  return rows;
}

export function formatUsdMicros(micros: number): string {
  if (!Number.isFinite(micros) || micros <= 0) {
    return formatUsdCents(0);
  }
  const dollars = micros / MICROS_PER_DOLLAR;
  if (dollars >= 0.01) {
    return `$${dollars.toFixed(2)}`;
  }
  return `$${trimDollarFraction(dollars.toFixed(6))}`;
}

/** Spreadsheet dollar amount from micros. Zero stays empty. */
export function formatUsageCsvDollarsFromMicros(micros: number): string {
  if (!Number.isFinite(micros) || micros <= 0) {
    return "";
  }
  const dollars = micros / MICROS_PER_DOLLAR;
  if (dollars >= 0.01) {
    return dollars.toFixed(2);
  }
  return trimDollarFraction(dollars.toFixed(6));
}

function trimDollarFraction(value: string): string {
  const trimmed = value.replace(/0+$/, "").replace(/\.$/, "");
  const dot = trimmed.indexOf(".");
  if (dot === -1 || trimmed.length - dot - 1 >= 2) {
    return trimmed;
  }
  return Number(trimmed).toFixed(2);
}

export function formatDurationSeconds(seconds: number): string {
  const safeSeconds = Number.isFinite(seconds) && seconds > 0 ? seconds : 0;
  if (safeSeconds < 60) {
    return `${safeSeconds} s`;
  }
  const totalMinutes = Math.floor(safeSeconds / 60);
  if (totalMinutes < 60) {
    const rest = safeSeconds % 60;
    if (rest === 0) {
      return `${totalMinutes} min`;
    }
    return `${totalMinutes} min ${rest} s`;
  }
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (minutes === 0) {
    return `${hours} h`;
  }
  return `${hours} h ${minutes} min`;
}

export function formatUsageTaskName(workOrderKey: string | undefined, title: string | undefined): string {
  const key = workOrderKey?.trim() ?? "";
  const name = title?.trim() || "Untitled task";
  return key ? `${key} · ${name}` : name;
}

export function formatUsageTokensAndTime(tokens: number, durationSeconds: number): string {
  const parts: string[] = [];
  if (tokens > 0) {
    parts.push(formatCompactTokens(tokens));
  }
  if (durationSeconds > 0) {
    parts.push(formatDurationSeconds(durationSeconds));
  }
  return parts.length > 0 ? parts.join(" · ") : "—";
}

export function formatUsageModels(models?: string[], byokModels?: string[]): string {
  return usageModelLabel(models, byokModels) || "—";
}

export function formatUsageMachineTypes(machineTypes?: string[]): string {
  return uniqueUsageLabels(machineTypes).join(" · ") || "—";
}

export function formatUsageTaskKey(workOrderKey: string | undefined): string {
  return workOrderKey?.trim() || "Untitled task";
}

/** Notes your keys only on the models your keys paid for, so a mixed run stays accurate. */
function usageModelLabel(models: string[] | undefined, byokModels: string[] | undefined): string {
  const hosted = uniqueUsageLabels(models);
  const byok = uniqueUsageLabels(byokModels);
  if (byok.length === 0) {
    return hosted.join(" · ");
  }
  const byokLabel = `${byok.join(" · ")} (your keys)`;
  if (hosted.length === 0) {
    return byokLabel;
  }
  return `${hosted.join(" · ")} · ${byokLabel}`;
}

export function formatUsageOccurredAt(value: string | undefined): string {
  if (!value) {
    return "—";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return "—";
  }
  const date = parsed.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    timeZone: "UTC",
  });
  const time = parsed
    .toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
      timeZone: "UTC",
    })
    .replace(/\u202f/g, " ");
  return `${date}, ${time}`;
}

function uniqueUsageLabels(values: string[] | undefined): string[] {
  const seen = new Set<string>();
  const labels: string[] = [];
  for (const value of values ?? []) {
    const label = machineTypeLabel(value.trim());
    if (!label || seen.has(label)) {
      continue;
    }
    seen.add(label);
    labels.push(label);
  }
  return labels;
}

export function formatWorkOrderUsage(totalTokens: number, totalCostCents: number, durationSeconds = 0): string | null {
  const parts: string[] = [];
  if (totalCostCents > 0) {
    parts.push(formatUsdCents(totalCostCents));
  }
  if (totalTokens > 0) {
    parts.push(formatCompactTokens(totalTokens));
  }
  if (durationSeconds > 0) {
    parts.push(formatDurationSeconds(durationSeconds));
  }
  return parts.length > 0 ? parts.join(" · ") : null;
}

interface WorkOrderExecutionUsage {
  totalTokens?: string | number;
  costCents?: string | number;
  durationSeconds?: string | number;
}

export function formatWorkOrderExecutionUsage(executions: WorkOrderExecutionUsage[]): string | null {
  let totalTokens = 0;
  let totalCostCents = 0;
  let durationSeconds = 0;
  for (const execution of executions) {
    totalTokens += parseWorkOrderMetric(execution.totalTokens);
    totalCostCents += parseWorkOrderMetric(execution.costCents);
    durationSeconds += parseWorkOrderMetric(execution.durationSeconds);
  }
  return formatWorkOrderUsage(totalTokens, totalCostCents, durationSeconds);
}

import { formatCompactTokenLabel } from "@/lib/formatTokenCount";

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

export function formatUsdCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
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

export interface UsageRunResources {
  /** Models the hosted credit paid for. */
  models?: string[];
  /** Models your own provider keys paid for. */
  byokModels?: string[];
  machineTypes?: string[];
}

export function formatUsageRunResources({ models, byokModels, machineTypes }: UsageRunResources): string {
  const parts = [usageModelLabel(models, byokModels), uniqueUsageLabels(machineTypes).join(" · ")].filter(Boolean);
  return parts.length > 0 ? parts.join(" · ") : "—";
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

export function formatUsageOccurredAtUtc(value: string | undefined): string {
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
    year: "numeric",
    timeZone: "UTC",
  });
  const hours = String(parsed.getUTCHours()).padStart(2, "0");
  const minutes = String(parsed.getUTCMinutes()).padStart(2, "0");
  return `${date} ${hours}:${minutes} UTC`;
}

function uniqueUsageLabels(values: string[] | undefined): string[] {
  const seen = new Set<string>();
  const labels: string[] = [];
  for (const value of values ?? []) {
    const label = value.trim();
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

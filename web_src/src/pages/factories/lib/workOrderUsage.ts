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

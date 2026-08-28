import { formatCompactTokenLabel } from "@/lib/formatTokenCount";

export function parseWorkOrderMetric(value: string | number | undefined): number {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function formatCompactTokens(tokens: number): string {
  return formatCompactTokenLabel(tokens);
}

export function formatUsdCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

export function formatWorkOrderUsage(totalTokens: number, totalCostCents: number): string | null {
  const parts: string[] = [];
  if (totalCostCents > 0) {
    parts.push(formatUsdCents(totalCostCents));
  }
  if (totalTokens > 0) {
    parts.push(formatCompactTokens(totalTokens));
  }
  return parts.length > 0 ? parts.join(" · ") : null;
}

interface WorkOrderExecutionUsage {
  totalTokens?: string | number;
  costCents?: string | number;
}

export function formatWorkOrderExecutionUsage(executions: WorkOrderExecutionUsage[]): string | null {
  let totalTokens = 0;
  let totalCostCents = 0;
  for (const execution of executions) {
    totalTokens += parseWorkOrderMetric(execution.totalTokens);
    totalCostCents += parseWorkOrderMetric(execution.costCents);
  }
  return formatWorkOrderUsage(totalTokens, totalCostCents);
}

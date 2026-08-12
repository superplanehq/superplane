export function parseWorkOrderMetric(value: string | number | undefined): number {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed : 0;
}

export function formatCompactTokens(tokens: number): string {
  if (tokens >= 1_000_000) {
    return `${(tokens / 1_000_000).toFixed(1)}M tokens`;
  }
  if (tokens >= 1_000) {
    return `${(tokens / 1_000).toFixed(0)}k tokens`;
  }
  return `${tokens} tokens`;
}

export function formatUsdCents(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`;
}

export function formatWorkOrderUsage(totalTokens: number, totalCostCents: number): string | null {
  const parts: string[] = [];
  if (totalTokens > 0) {
    parts.push(formatCompactTokens(totalTokens));
  }
  if (totalCostCents > 0) {
    parts.push(formatUsdCents(totalCostCents));
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

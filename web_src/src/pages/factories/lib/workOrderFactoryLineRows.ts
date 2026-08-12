import type { FactoriesWorkOrderExecution } from "@/api-client";

export type FactoryLineRowTone = "success" | "warning" | "danger" | "muted";

export interface FactoryLineRowModel {
  lineId: string;
  lineName: string;
  tone: FactoryLineRowTone;
}

// Groups a work order's executions by factory line, collapsing multiple runs
// on the same line into the worst tone so the sidebar surfaces failures over
// successes.
export function deriveFactoryLineRows(executions: FactoriesWorkOrderExecution[]): FactoryLineRowModel[] {
  const map = new Map<string, FactoryLineRowModel>();
  for (const execution of executions) {
    const lineId = execution.line?.id ?? "unknown";
    const lineName = execution.line?.name?.trim() || "Unnamed line";
    const tone = executionTone(execution);
    const existing = map.get(lineId);
    if (!existing) {
      map.set(lineId, { lineId, lineName, tone });
      continue;
    }
    existing.tone = worstTone(existing.tone, tone);
  }
  return [...map.values()];
}

function executionTone(execution: FactoriesWorkOrderExecution): FactoryLineRowTone {
  if (execution.result === "RESULT_FAILED") return "danger";
  if (
    execution.state === "STATE_PENDING" ||
    execution.state === "STATE_STARTED" ||
    execution.state === "STATE_CANCELLING"
  ) {
    return "warning";
  }
  if (execution.result === "RESULT_PASSED") return "success";
  return "muted";
}

const TONE_PRIORITY: Record<FactoryLineRowTone, number> = {
  danger: 3,
  warning: 2,
  success: 1,
  muted: 0,
};

function worstTone(a: FactoryLineRowTone, b: FactoryLineRowTone): FactoryLineRowTone {
  return TONE_PRIORITY[a] >= TONE_PRIORITY[b] ? a : b;
}

export const FACTORY_LINE_TONE_DOT_CLASS: Record<FactoryLineRowTone, string> = {
  success: "bg-[var(--status-success-dot)]",
  warning: "bg-[var(--status-warning-dot)]",
  danger: "bg-[var(--status-danger-dot)]",
  muted: "bg-muted-foreground/40",
};

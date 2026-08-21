import type { FactoriesWorkOrderLineDispatch } from "@/api-client";

export type FactoryLineRowTone = "success" | "warning" | "danger" | "muted";

export interface FactoryLineRowModel {
  lineId: string;
  lineName: string;
  tone: FactoryLineRowTone;
}

// One row per line, summarizing that line's most recent dispatch. The
// sidebar is a compact status summary, not a history view — every dispatch
// (including earlier, superseded ones) shows up separately in
// WorkOrderExecutionsList instead.
export function deriveFactoryLineRows(dispatches: FactoriesWorkOrderLineDispatch[]): FactoryLineRowModel[] {
  const latestByLineId = new Map<string, { model: FactoryLineRowModel; createdAt: number }>();

  for (const dispatch of dispatches) {
    const lineId = dispatch.line?.id ?? "unknown";
    const lineName = dispatch.line?.name?.trim() || "Unnamed line";
    const createdAt = Date.parse(dispatch.createdAt ?? "") || 0;

    const existing = latestByLineId.get(lineId);
    if (existing && existing.createdAt > createdAt) {
      continue;
    }

    latestByLineId.set(lineId, {
      model: { lineId, lineName, tone: dispatchTone(dispatch) },
      createdAt,
    });
  }

  return [...latestByLineId.values()].map(({ model }) => model);
}

function dispatchTone(dispatch: FactoriesWorkOrderLineDispatch): FactoryLineRowTone {
  if (dispatch.result === "RESULT_FAILED") return "danger";
  if (dispatch.state === "STATE_ACTIVE") return "warning";
  if (dispatch.result === "RESULT_PASSED") return "success";
  return "muted";
}

export const FACTORY_LINE_TONE_DOT_CLASS: Record<FactoryLineRowTone, string> = {
  success: "bg-[var(--status-success-dot)]",
  warning: "bg-[var(--status-warning-dot)]",
  danger: "bg-[var(--status-danger-dot)]",
  muted: "bg-muted-foreground/40",
};

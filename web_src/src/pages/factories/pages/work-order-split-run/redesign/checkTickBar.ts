import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";

const MAX_TICKS = 10;
const RATIO_TICKS = 20;

/** One box per score tick when the scale is small; a dense bar for percents. */
export function checkTickBar(check: Pick<WorkOrderCheckPresentation, "score" | "maxScore" | "format">): {
  total: number;
  filled: number;
} {
  if (check.format === "boolean") {
    return { total: 1, filled: check.score > 0 ? 1 : 0 };
  }
  if (check.format === "percent") {
    return { total: RATIO_TICKS, filled: clampTicks(Math.round((check.score / 100) * RATIO_TICKS), RATIO_TICKS) };
  }
  const max = check.maxScore;
  if (max > 0 && max <= MAX_TICKS) {
    return { total: max, filled: clampTicks(Math.round(check.score), max) };
  }
  if (max > 0) {
    return { total: RATIO_TICKS, filled: clampTicks(Math.round((check.score / max) * RATIO_TICKS), RATIO_TICKS) };
  }
  return { total: 0, filled: 0 };
}

function clampTicks(filled: number, total: number): number {
  return Math.min(Math.max(filled, 0), total);
}

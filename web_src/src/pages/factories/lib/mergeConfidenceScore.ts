import { workOrderCheckStatus, type WorkOrderCheckLevel, type WorkOrderCheckPresentation } from "./workOrderChecks";

/** Check keys written by the merge confidence automation. */
export const MERGE_CONFIDENCE_METRIC_KEYS = [
  "risk-review",
  "performance-review",
  "security-review",
  "drift-review",
  "reversibility-review",
] as const;

const LOWER_IS_BETTER_KEYS = new Set<string>(["risk-review", "drift-review"]);

const LEVEL_SEVERITY: Record<WorkOrderCheckLevel, number> = {
  positive: 0,
  neutral: 1,
  caution: 2,
  critical: 3,
};

export const MERGE_CONFIDENCE_SCORE_NAME = "Merge confidence";

/** Light wash in the same hue as the score, so the panel matches the bar. */
const PANEL_TONE: Record<string, string> = {
  High: "[--frame-panel-bg:var(--color-emerald-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-emerald-500)_16%,var(--color-card))]",
  Healthy:
    "[--frame-panel-bg:var(--color-emerald-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-emerald-500)_16%,var(--color-card))]",
  Medium:
    "[--frame-panel-bg:var(--color-orange-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-orange-500)_16%,var(--color-card))]",
  Caution:
    "[--frame-panel-bg:var(--color-orange-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-orange-500)_16%,var(--color-card))]",
  Low: "[--frame-panel-bg:var(--color-red-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-red-500)_16%,var(--color-card))]",
  Critical:
    "[--frame-panel-bg:var(--color-red-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-red-500)_16%,var(--color-card))]",
  Neutral:
    "[--frame-panel-bg:var(--color-slate-50)] dark:[--frame-panel-bg:color-mix(in_oklch,var(--color-slate-400)_16%,var(--color-card))]",
};

export function mergeConfidencePanelTone(check: Pick<WorkOrderCheckPresentation, "name" | "score" | "level">): string {
  return PANEL_TONE[workOrderCheckStatus(check).label] ?? PANEL_TONE.Neutral;
}

export interface MergeConfidenceCheckGroup {
  check: WorkOrderCheckPresentation;
  metrics: WorkOrderCheckPresentation[];
}

export function isMergeConfidenceMetric(check: Pick<WorkOrderCheckPresentation, "key">): boolean {
  return check.key != null && (MERGE_CONFIDENCE_METRIC_KEYS as readonly string[]).includes(check.key);
}

/**
 * The task summary shows merge confidence only. Planning confidence and other
 * checks stay off this list once work has started.
 */
export function consoleCheckList(checks: WorkOrderCheckPresentation[]): MergeConfidenceCheckGroup | null {
  const metrics = orderMergeConfidenceMetrics(checks.filter(isMergeConfidenceMetric));
  if (metrics.length === 0) {
    return null;
  }
  return { check: mergeConfidenceScore(metrics), metrics };
}

/** Higher is better. Risk and drift are flipped so 5 means the change is safe to merge. */
export function mergeConfidenceScore(metrics: WorkOrderCheckPresentation[]): WorkOrderCheckPresentation {
  const maxScore = metrics[0]?.maxScore ?? 0;
  const weakest = metrics.reduce((lowest, check) => Math.min(lowest, confidenceRatio(check)), 1);
  return {
    id: "merge-confidence",
    key: "merge-confidence",
    name: MERGE_CONFIDENCE_SCORE_NAME,
    score: maxScore > 0 ? Math.round(weakest * maxScore) : 0,
    maxScore,
    format: "fraction",
    level: weakestLevel(metrics),
  };
}

function orderMergeConfidenceMetrics(metrics: WorkOrderCheckPresentation[]): WorkOrderCheckPresentation[] {
  return [...metrics].sort((left, right) => metricIndex(left.key) - metricIndex(right.key));
}

function metricIndex(key: string | undefined): number {
  const index = MERGE_CONFIDENCE_METRIC_KEYS.indexOf(key as (typeof MERGE_CONFIDENCE_METRIC_KEYS)[number]);
  return index === -1 ? MERGE_CONFIDENCE_METRIC_KEYS.length : index;
}

function confidenceRatio(check: WorkOrderCheckPresentation): number {
  const maxScore = check.maxScore;
  if (maxScore <= 0) {
    return 0;
  }
  const points = check.key != null && LOWER_IS_BETTER_KEYS.has(check.key) ? maxScore + 1 - check.score : check.score;
  return Math.min(maxScore, Math.max(0, points)) / maxScore;
}

function weakestLevel(metrics: WorkOrderCheckPresentation[]): WorkOrderCheckLevel {
  return metrics.reduce<WorkOrderCheckLevel>((weakest, check) => {
    return LEVEL_SEVERITY[check.level] > LEVEL_SEVERITY[weakest] ? check.level : weakest;
  }, "positive");
}

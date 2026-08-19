import type { FactoriesWorkOrderCheck, WorkOrderCheckLevel as ApiWorkOrderCheckLevel } from "@/api-client";

/** How strongly the reported score should alarm (or reassure) the reader.
 * The emitting automation decides — the UI cannot know whether a high
 * number is good (coverage) or bad (risk). */
export type WorkOrderCheckLevel = "positive" | "neutral" | "caution" | "critical";

interface WorkOrderCheckPresentationBase {
  id: string;
  /** Short human name, e.g. "Risk review" or "CI". */
  name: string;
  level: WorkOrderCheckLevel;
  /** One-line result, shown in the expanded dialog under the score/badge. */
  summary?: string;
  /** Full markdown analysis behind the score/badge. */
  analysis?: string;
  /** Automation that produced the check, e.g. "PR Risk Review". */
  sourceName?: string;
  /** App + run that reported the check — powers the "View run" link. */
  appId?: string;
  runId?: string;
  updatedAt?: string;
}

export interface ScoreCheckPresentation extends WorkOrderCheckPresentationBase {
  /** Absent (or "score") means the check reports a numeric score. */
  type?: "score";
  score: number;
  maxScore: number;
  /** "percent" renders `82%`; "fraction" (default) renders `65/100`. */
  format?: "fraction" | "percent";
  /** Score from the previous report of the same check — powers the trend delta. */
  previousScore?: number;
}

export interface BooleanCheckPresentation extends WorkOrderCheckPresentationBase {
  /** Gate-style check that reports pass/fail instead of a score — no
   * maxScore, format, or thresholds. Renders as a Pass/Fail badge. */
  type: "boolean";
  passed: boolean;
  /** Previous pass/fail state, when known — powers a "Pass → Fail" timeline phrasing. */
  previousPassed?: boolean;
}

/** A check is either a numeric score (default) or a pass/fail boolean. */
export type WorkOrderCheckPresentation = ScoreCheckPresentation | BooleanCheckPresentation;

export function isBooleanCheck(check: WorkOrderCheckPresentation): check is BooleanCheckPresentation {
  return check.type === "boolean";
}

/** Textual verdict next to the score — color alone must not carry the meaning
 * (see Vercel Speed Insights / Shopify fraud analysis). */
export const LEVEL_LABEL: Record<WorkOrderCheckLevel, { label: string; className: string; badgeClassName: string }> = {
  positive: {
    label: "Healthy",
    className: "text-emerald-700 dark:text-emerald-400",
    badgeClassName: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  },
  neutral: {
    label: "Neutral",
    className: "text-slate-600 dark:text-slate-400",
    badgeClassName: "border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-400",
  },
  caution: {
    label: "Needs attention",
    className: "text-amber-700 dark:text-amber-400",
    badgeClassName: "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400",
  },
  critical: {
    label: "Critical",
    className: "text-red-700 dark:text-red-400",
    badgeClassName: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
  },
};

export function formatCheckScore(check: Pick<ScoreCheckPresentation, "score" | "maxScore" | "format">): {
  value: string;
  scale: string;
} {
  if (check.format === "percent") {
    return { value: String(check.score), scale: "%" };
  }
  return { value: String(check.score), scale: `/${check.maxScore}` };
}

const LEVEL_FROM_API: Partial<Record<ApiWorkOrderCheckLevel, WorkOrderCheckLevel>> = {
  LEVEL_POSITIVE: "positive",
  LEVEL_NEUTRAL: "neutral",
  LEVEL_CAUTION: "caution",
  LEVEL_CRITICAL: "critical",
};

function presentCheckLevel(level: ApiWorkOrderCheckLevel | undefined): WorkOrderCheckLevel {
  return (level && LEVEL_FROM_API[level]) || "neutral";
}

function emptyToUndefined(value: string | undefined): string | undefined {
  return value || undefined;
}

export function presentWorkOrderCheck(check: FactoriesWorkOrderCheck): ScoreCheckPresentation {
  const automation = check.automation;
  return {
    id: check.id ?? check.key ?? "",
    name: check.name ?? "",
    score: check.score ?? 0,
    maxScore: check.maxScore ?? 0,
    format: check.format === "FORMAT_PERCENT" ? "percent" : "fraction",
    level: presentCheckLevel(check.level),
    previousScore: check.previousScore,
    summary: emptyToUndefined(check.summary),
    analysis: emptyToUndefined(check.analysis),
    sourceName: emptyToUndefined(automation?.appName) ?? emptyToUndefined(automation?.nodeName),
    appId: emptyToUndefined(automation?.appId),
    runId: emptyToUndefined(check.runId),
    updatedAt: check.updatedAt,
  };
}

export function presentWorkOrderChecks(checks: FactoriesWorkOrderCheck[]): WorkOrderCheckPresentation[] {
  return checks.map(presentWorkOrderCheck);
}

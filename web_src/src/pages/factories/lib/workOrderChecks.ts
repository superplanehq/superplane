import type { FactoriesWorkOrderCheck, WorkOrderCheckLevel as ApiWorkOrderCheckLevel } from "@/api-client";

import { CONFIDENCE_CHECK_NAME, confidenceBandForScore, type ConfidenceBand } from "./confidenceScore";

/** How strongly the reported score should alarm (or reassure) the reader.
 * The emitting automation decides — the UI cannot know whether a high
 * number is good (coverage) or bad (risk). */
export type WorkOrderCheckLevel = "positive" | "neutral" | "caution" | "critical";

/** How the score renders: `65/100`, `82%`, or a Pass/Fail verdict. */
export type WorkOrderCheckScoreFormat = "fraction" | "percent" | "boolean";

export interface WorkOrderCheckPresentation {
  id: string;
  /** Short human name, e.g. "Risk score" or "Code quality". */
  name: string;
  score: number;
  maxScore: number;
  /** "percent" renders `82%`; "boolean" renders `Pass`/`Fail`;
   * "fraction" (default) renders `65/100`. */
  format?: WorkOrderCheckScoreFormat;
  level: WorkOrderCheckLevel;
  /** Score from the previous report of the same check — powers the trend delta. */
  previousScore?: number;
  /** Recent run scores, oldest → newest, ending with the current score —
   * powers the segmented run-history strip on boolean checks. */
  recentScores?: number[];
  /** One-line result, shown in the expanded dialog under the score. */
  summary?: string;
  /** Full markdown analysis behind the score. */
  analysis?: string;
  /** Automation that produced the check, e.g. "PR Risk Review". */
  sourceName?: string;
  /** App + run that reported the score — powers the "View run" link. */
  appId?: string;
  runId?: string;
  updatedAt?: string;
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

const CONFIDENCE_BAND_LABEL: Record<ConfidenceBand, (typeof LEVEL_LABEL)[WorkOrderCheckLevel]> = {
  High: {
    label: "High",
    className: "text-emerald-700 dark:text-emerald-400",
    badgeClassName: "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  },
  Medium: {
    label: "Medium",
    className: "text-orange-700 dark:text-orange-400",
    badgeClassName: "border-orange-500/30 bg-orange-500/10 text-orange-700 dark:text-orange-400",
  },
  Low: {
    label: "Low",
    className: "text-red-700 dark:text-red-400",
    badgeClassName: "border-red-500/30 bg-red-500/10 text-red-700 dark:text-red-400",
  },
};

/** Confidence uses High / Medium / Low. Other checks use Healthy / Needs attention. */
export function workOrderCheckStatus(check: Pick<WorkOrderCheckPresentation, "name" | "score" | "level">) {
  if (check.name === CONFIDENCE_CHECK_NAME) {
    return CONFIDENCE_BAND_LABEL[confidenceBandForScore(check.score)];
  }
  return LEVEL_LABEL[check.level];
}

/** Verdict text for a boolean (pass/fail) check score. */
export function booleanCheckVerdict(score: number): "Pass" | "Fail" {
  return score > 0 ? "Pass" : "Fail";
}

export function formatCheckScore(check: Pick<WorkOrderCheckPresentation, "score" | "maxScore" | "format">): {
  value: string;
  scale: string;
} {
  if (check.format === "boolean") {
    return { value: booleanCheckVerdict(check.score), scale: "" };
  }
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

function presentCheckFormat(format: FactoriesWorkOrderCheck["format"]): WorkOrderCheckScoreFormat {
  switch (format) {
    case "FORMAT_PERCENT":
      return "percent";
    case "FORMAT_BOOLEAN":
      return "boolean";
    default:
      return "fraction";
  }
}

function emptyToUndefined(value: string | undefined): string | undefined {
  return value || undefined;
}

export function presentWorkOrderCheck(check: FactoriesWorkOrderCheck): WorkOrderCheckPresentation {
  const automation = check.automation;
  return {
    id: check.id ?? check.key ?? "",
    name: check.name ?? "",
    score: check.score ?? 0,
    maxScore: check.maxScore ?? 0,
    format: presentCheckFormat(check.format),
    level: presentCheckLevel(check.level),
    previousScore: check.previousScore,
    recentScores: check.recentScores,
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

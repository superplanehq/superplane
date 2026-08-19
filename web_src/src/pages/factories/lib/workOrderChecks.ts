import type { FactoriesWorkOrderCheck, WorkOrderCheckLevel as ApiWorkOrderCheckLevel } from "@/api-client";

/** How strongly the reported score should alarm (or reassure) the reader.
 * The emitting automation decides — the UI cannot know whether a high
 * number is good (coverage) or bad (risk). */
export type WorkOrderCheckLevel = "positive" | "neutral" | "caution" | "critical";

/** How the score renders: `65/100`, `82%`, or a Pass/Fail verdict. */
export type WorkOrderCheckScoreFormat = "fraction" | "percent" | "boolean";

export interface WorkOrderCheckPresentation {
  id: string;
  /** Short human name, e.g. "Risk review" or "Code coverage". */
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
  // Widened: the SDK union gains FORMAT_BOOLEAN once the backend proto lands
  // and `make pb.gen` regenerates the client.
  switch (format as string) {
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
    // Widened: the SDK gains recentScores with the boolean-checks backend
    // change and a `make pb.gen` regeneration.
    recentScores: (check as { recentScores?: number[] }).recentScores,
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

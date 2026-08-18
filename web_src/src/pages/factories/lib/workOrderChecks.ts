import type { FactoriesWorkOrderCheck, WorkOrderCheckLevel as ApiWorkOrderCheckLevel } from "@/api-client";

/** How strongly the reported score should alarm (or reassure) the reader.
 * The emitting automation decides — the UI cannot know whether a high
 * number is good (coverage) or bad (risk). */
export type WorkOrderCheckLevel = "positive" | "neutral" | "caution" | "critical";

export interface WorkOrderCheckPresentation {
  id: string;
  /** Short human name, e.g. "Risk review" or "Code coverage". */
  name: string;
  score: number;
  maxScore: number;
  /** "percent" renders `82%`; "fraction" (default) renders `65/100`. */
  format?: "fraction" | "percent";
  level: WorkOrderCheckLevel;
  /** Score from the previous report of the same check — powers the trend delta. */
  previousScore?: number;
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

const LEVEL_FROM_API: Partial<Record<ApiWorkOrderCheckLevel, WorkOrderCheckLevel>> = {
  LEVEL_POSITIVE: "positive",
  LEVEL_NEUTRAL: "neutral",
  LEVEL_CAUTION: "caution",
  LEVEL_CRITICAL: "critical",
};

export function presentWorkOrderCheck(check: FactoriesWorkOrderCheck): WorkOrderCheckPresentation {
  return {
    id: check.id ?? check.key ?? "",
    name: check.name ?? "",
    score: check.score ?? 0,
    maxScore: check.maxScore ?? 0,
    format: check.format === "FORMAT_PERCENT" ? "percent" : "fraction",
    level: (check.level && LEVEL_FROM_API[check.level]) || "neutral",
    previousScore: check.previousScore,
    summary: check.summary || undefined,
    analysis: check.analysis || undefined,
    sourceName: check.automation?.appName || check.automation?.nodeName || undefined,
    appId: check.automation?.appId || undefined,
    runId: check.runId || undefined,
    updatedAt: check.updatedAt,
  };
}

export function presentWorkOrderChecks(checks: FactoriesWorkOrderCheck[]): WorkOrderCheckPresentation[] {
  return checks.map(presentWorkOrderCheck);
}

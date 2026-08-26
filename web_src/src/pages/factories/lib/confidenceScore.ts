import type { WorkOrderCheckLevel } from "./workOrderChecks";
import type { WorkOrderDisplayStatus } from "./workOrderProgress";

export const CONFIDENCE_SCORE_MAX = 5;

export const CONFIDENCE_CHECK_NAME = "Confidence score";

/** Board cards show the meter next to Start. Only drafts need the request. */
export function boardCardLoadsConfidenceChecks(displayStatus: WorkOrderDisplayStatus): boolean {
  return displayStatus === "draft";
}

export function confidenceScoreFromChecks(
  checks: Array<{ name?: string; score?: number }> | undefined,
): number | undefined {
  const check = (checks ?? []).find((entry) => entry.name === CONFIDENCE_CHECK_NAME);
  if (check?.score == null) {
    return undefined;
  }
  return clampConfidenceScore(check.score);
}

/** Intake scores arrive as a percentage. The meter shows five steps. */
export function confidenceScoreFromPercent(percent: number): number {
  return clampConfidenceScore((percent / 100) * CONFIDENCE_SCORE_MAX);
}

export type ConfidenceBand = "High" | "Medium" | "Low";

export function clampConfidenceScore(score: number): number {
  return Math.min(CONFIDENCE_SCORE_MAX, Math.max(0, Math.round(score)));
}

export function confidenceBandForScore(score: number): ConfidenceBand {
  if (score >= 4) {
    return "High";
  }
  if (score >= 3) {
    return "Medium";
  }
  return "Low";
}

export function confidenceCheckLevel(score: number): WorkOrderCheckLevel {
  if (score >= 4) {
    return "positive";
  }
  if (score >= 3) {
    return "neutral";
  }
  return "caution";
}

/** One-line result: how suitable the source issue is for an agent on this line. */
export function confidenceSuitabilitySummary(band: ConfidenceBand): string {
  if (band === "High") {
    return "This issue is a good fit for an agent on this factory line.";
  }
  if (band === "Medium") {
    return "This issue is a mixed fit for an agent on this factory line.";
  }
  return "This issue is a poor fit for an agent on this factory line.";
}

/** Analysis: SuperPlane read the source issue and scored agent suitability. */
export function confidenceSuitabilityAnalysis(params: { source?: string; reasons?: readonly string[] }): string {
  const source = params.source?.trim();
  const intro = source
    ? `The automation read this ${source} issue. It scored how suitable the work is for an agent on this factory line.`
    : "The automation read this issue. It scored how suitable the work is for an agent on this factory line.";
  const reasons = params.reasons ?? [];
  if (reasons.length === 0) {
    return intro;
  }
  return [intro, "", "### Why this score", ...reasons.map((reason) => `- ${reason}`)].join("\n");
}

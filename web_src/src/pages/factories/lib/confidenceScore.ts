import type { WorkOrderCheckLevel } from "./workOrderChecks";
import type { WorkOrderDisplayStatus } from "./workOrderProgress";

export const CONFIDENCE_SCORE_MAX = 5;

/** Clarity: how well the task is defined. Written by the refine session. */
export const CLARITY_CHECK_KEY = "clarity";
export const CLARITY_CHECK_NAME = "Clarity score";
/** Confidence: how likely an agent finishes the task in one run. Written by refine and intake. */
export const CONFIDENCE_CHECK_KEY = "confidence";
export const CONFIDENCE_CHECK_NAME = "Confidence score";

export const SCORE_CHECK_NAMES: readonly string[] = [CLARITY_CHECK_NAME, CONFIDENCE_CHECK_NAME];

export function isScoreCheckName(name: string | undefined): boolean {
  return name != null && SCORE_CHECK_NAMES.includes(name);
}

/** Board cards show the meter next to Start. Only drafts need the request. */
export function boardCardLoadsConfidenceChecks(displayStatus: WorkOrderDisplayStatus): boolean {
  return displayStatus === "draft";
}

type ScoreCheckLike = { name?: string; score?: number };

export function scoreFromChecks(checks: ScoreCheckLike[] | undefined, name: string): number | undefined {
  const check = (checks ?? []).find((entry) => entry.name === name);
  if (check?.score == null) {
    return undefined;
  }
  return clampConfidenceScore(check.score);
}

export function clarityScoreFromChecks(checks: ScoreCheckLike[] | undefined): number | undefined {
  return scoreFromChecks(checks, CLARITY_CHECK_NAME);
}

export function confidenceScoreFromChecks(checks: ScoreCheckLike[] | undefined): number | undefined {
  return scoreFromChecks(checks, CONFIDENCE_CHECK_NAME);
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

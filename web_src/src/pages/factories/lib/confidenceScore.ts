import type { WorkOrderCheckLevel } from "./workOrderChecks";

export const CONFIDENCE_SCORE_MAX = 5;

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

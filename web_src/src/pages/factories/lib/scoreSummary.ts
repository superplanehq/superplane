export type ScoreKind = "clarity" | "confidence";

export type ScoreSummaryValue = {
  score?: number;
  summary?: string;
};

export const SCORE_KINDS: readonly ScoreKind[] = ["clarity", "confidence"];

/** Shown when the agent published a score without a summary. */
export const SCORE_FALLBACK_SUMMARY: Record<ScoreKind, string> = {
  clarity: "The analysis scored how clear this work is.",
  confidence: "The analysis scored how likely an agent finishes this work in one run.",
};

export function scoreSummaryText(kind: ScoreKind, value?: ScoreSummaryValue): string | undefined {
  if (value?.score == null) {
    return undefined;
  }
  return value.summary?.trim() || SCORE_FALLBACK_SUMMARY[kind];
}

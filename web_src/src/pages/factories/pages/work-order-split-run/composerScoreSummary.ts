import type { RefineSummaryKind } from "./refineLayoutPreference";

export type ComposerScore = {
  score?: number;
  summary?: string;
};

export type ComposerSummaryBodies = Record<RefineSummaryKind, string | undefined>;

export const SCORE_FALLBACK_WHY: Record<RefineSummaryKind, string> = {
  clarity: "The analysis scored how clear this work is.",
  confidence: "The analysis scored how likely an agent finishes this work in one run.",
};

const OTHER_SUMMARY: Record<RefineSummaryKind, RefineSummaryKind> = {
  clarity: "confidence",
  confidence: "clarity",
};

function summaryBody(kind: RefineSummaryKind, value: ComposerScore | undefined): string | undefined {
  if (value?.score == null) {
    return undefined;
  }
  return value.summary?.trim() || SCORE_FALLBACK_WHY[kind];
}

export function composerSummaryBodies(clarity?: ComposerScore, confidence?: ComposerScore): ComposerSummaryBodies {
  return {
    clarity: summaryBody("clarity", clarity),
    confidence: summaryBody("confidence", confidence),
  };
}

/**
 * The stored preference names a kind. When that score has no summary yet
 * (for example an intake-only draft has Confidence but no Clarity), show
 * the one that exists instead of an empty drawer.
 */
export function resolveOpenSummary(
  preferred: RefineSummaryKind | null,
  bodies: ComposerSummaryBodies,
): RefineSummaryKind | null {
  if (preferred === null || bodies[preferred]) {
    return preferred;
  }
  const other = OTHER_SUMMARY[preferred];
  return bodies[other] ? other : preferred;
}

/**
 * A click on the open chip closes the drawer. A click on the other chip
 * moves the drawer to that score. Returns the kind to pass to the toggle.
 */
export function summaryToggleTarget(
  clicked: RefineSummaryKind,
  shown: RefineSummaryKind | null,
  preferred: RefineSummaryKind | null,
): RefineSummaryKind {
  if (shown === clicked && preferred) {
    return preferred;
  }
  return clicked;
}

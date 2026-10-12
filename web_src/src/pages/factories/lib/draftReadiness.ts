/**
 * One verdict from the two refine scores. Clarity says how well the task is
 * defined; the user can fix a low Clarity in the chat. Confidence says how
 * likely an agent finishes the task in one run; a low Confidence is a risk the
 * user accepts or removes by splitting the work.
 */
export type DraftReadinessTone = "analyzing" | "pending" | "blocked" | "caution" | "ready";

export type DraftReadinessNote = {
  headline: string;
  text: string;
};

export type DraftReadiness = DraftReadinessNote & {
  tone: DraftReadinessTone;
};

export type DraftReadinessInput = {
  clarity?: number;
  confidence?: number;
  isAnalyzing?: boolean;
  /** Top of the scale the scores use: 5 for legacy scores, 3 for review scores. */
  scoreMax?: number;
};

const LOW_SCORE_MAX = 2;
const MID_SCORE = 3;

/** Legacy 1–5 scores: low is 1–2, the midpoint is 3. Review 1–3 scores: low is 1, the midpoint is 2. */
function scoreThresholds(scoreMax?: number): { low: number; mid: number } {
  if (scoreMax != null && scoreMax <= 3) {
    return { low: 1, mid: 2 };
  }
  return { low: LOW_SCORE_MAX, mid: MID_SCORE };
}

export const DRAFT_READINESS_NOTES = {
  analyzing: {
    headline: "SuperPlane is currently analyzing this task",
    text: "Wait for the analysis to finish. Or click Start to send this task to the line now.",
  },
  pending: {
    headline: "This task is ready to start",
    text: "Review the summary and the plan. Then click Start to send it to the line.",
  },
  unclear: {
    headline: "This task is not ready to start",
    text: "The task is not clear enough. Tell the agent more in the chat.",
  },
  agentFit: {
    headline: "Review before you start",
    text: "An agent may need steering. Start if you accept the risk, or split the work.",
  },
  uncertain: {
    headline: "Review the plan before you start",
    text: "The work is still uncertain. Add more context, or start if you accept the risk.",
  },
  ready: {
    headline: "This task is ready to start",
    text: "Review the summary and the plan. Then click Start to send it to the line.",
  },
} as const satisfies Record<string, DraftReadinessNote>;

/** One or two words for the board card. The full headline is in the tooltip. */
export const DRAFT_READINESS_SHORT_LABEL: Record<DraftReadinessTone, string> = {
  analyzing: "Analyzing",
  pending: "Ready",
  blocked: "Not ready",
  caution: "Review",
  ready: "Ready",
};

export type StartEmphasis = "filled" | "outline";

/** Start is filled only when the verdict says go. Other tones keep it available but quiet. */
export function startEmphasisForTone(tone: DraftReadinessTone): StartEmphasis {
  return tone === "ready" || tone === "pending" ? "filled" : "outline";
}

function withTone(tone: DraftReadinessTone, note: DraftReadinessNote): DraftReadiness {
  return { tone, ...note };
}

/** While the agent works the verdict says so, even when older scores exist. */
export function liveDraftReadiness(input: DraftReadinessInput): DraftReadiness {
  if (input.isAnalyzing) {
    return withTone("analyzing", DRAFT_READINESS_NOTES.analyzing);
  }
  return draftReadiness(input);
}

/** Intake drafts have Confidence only, so the verdict uses the scores that exist. */
export function draftReadiness(input: DraftReadinessInput): DraftReadiness {
  const { clarity, confidence } = input;
  if (clarity == null && confidence == null) {
    return input.isAnalyzing
      ? withTone("analyzing", DRAFT_READINESS_NOTES.analyzing)
      : withTone("pending", DRAFT_READINESS_NOTES.pending);
  }
  const { low, mid } = scoreThresholds(input.scoreMax);
  if (clarity != null && clarity <= low) {
    return withTone("blocked", DRAFT_READINESS_NOTES.unclear);
  }
  if (confidence != null && confidence <= low) {
    return withTone("caution", DRAFT_READINESS_NOTES.agentFit);
  }
  if (clarity === mid || confidence === mid) {
    return withTone("caution", DRAFT_READINESS_NOTES.uncertain);
  }
  return withTone("ready", DRAFT_READINESS_NOTES.ready);
}

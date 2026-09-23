import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";

export type PlanningSetupStep = "refine" | "clarity" | "confidence";

export type PlanningSetupPreviewScore = {
  kind: "clarity" | "confidence";
  score: number;
  summary?: string;
  emphasized: boolean;
};

/**
 * What the agent says in the chat pane.
 * - question: the agent reviewed the request, wrote a plan, and asks one planning question.
 * - confidence: the task is too big for one run; the agent offers a split.
 * - clarity: a few details are not clear; the agent asks for them.
 */
export type PlanningSetupPreviewChat = "question" | "confidence" | "clarity";

export type PlanningSetupPreviewScene = {
  body: "source" | "plan";
  chat: PlanningSetupPreviewChat;
  scores: PlanningSetupPreviewScore[];
  note: "ready" | "caution";
  showStart: boolean;
};

type PlanningSetupPreviewInput = {
  step: PlanningSetupStep;
  enabled: boolean;
  clarity: boolean;
  confidence: boolean;
};

const HIGH_SCORE = 4;
const MID_SCORE = 3;
const LOW_SCORE = 2;

export function planningSetupPreviewCaption(input: PlanningSetupPreviewInput): string {
  if (input.step === "refine") {
    return input.enabled
      ? PLANNING_SETTINGS_COPY.wizardPreviewRefineCaption
      : PLANNING_SETTINGS_COPY.wizardPreviewSourceCaption;
  }
  if (input.step === "clarity") {
    return input.clarity
      ? PLANNING_SETTINGS_COPY.wizardPreviewClarityOn
      : PLANNING_SETTINGS_COPY.wizardPreviewClarityOff;
  }
  return input.confidence
    ? PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOn
    : PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOff;
}

export function planningSetupPreviewScene(input: PlanningSetupPreviewInput): PlanningSetupPreviewScene {
  if (input.step === "refine" && !input.enabled) {
    return { body: "source", chat: "question", scores: [], note: "ready", showStart: true };
  }
  const lowConfidence = input.step === "confidence" && input.confidence;
  return {
    body: "plan",
    chat: planningSetupPreviewChat(input),
    scores: planningSetupPreviewScores(input),
    note: lowConfidence ? "caution" : "ready",
    showStart: false,
  };
}

function planningSetupPreviewChat(input: PlanningSetupPreviewInput): PlanningSetupPreviewChat {
  if (input.step === "confidence" && input.confidence) {
    return "confidence";
  }
  if (input.step === "clarity" && input.clarity) {
    return "clarity";
  }
  return "question";
}

/**
 * The Refine page shows the chat and the plan only. Confidence appears on its
 * page. Clarity appears on the last page next to a muted Confidence.
 */
function planningSetupPreviewScores(input: PlanningSetupPreviewInput): PlanningSetupPreviewScore[] {
  const scores: PlanningSetupPreviewScore[] = [];
  if (input.step === "refine") {
    return scores;
  }
  if (input.confidence) {
    const onConfidencePage = input.step === "confidence";
    scores.push({
      kind: "confidence",
      score: onConfidencePage ? LOW_SCORE : HIGH_SCORE,
      summary: onConfidencePage ? PLANNING_SETTINGS_COPY.wizardPreviewConfidenceSummary : undefined,
      emphasized: onConfidencePage,
    });
  }
  if (input.step === "clarity" && input.clarity) {
    scores.push({
      kind: "clarity",
      score: MID_SCORE,
      summary: PLANNING_SETTINGS_COPY.wizardPreviewClaritySummary,
      emphasized: true,
    });
  }
  return scores;
}

export function planningSetupParentStep(step: PlanningSetupStep): PlanningSetupStep | undefined {
  if (step === "clarity") {
    return "confidence";
  }
  if (step === "confidence") {
    return "refine";
  }
  return undefined;
}

export function planningSetupChildStep(step: PlanningSetupStep, enabled: boolean): PlanningSetupStep | undefined {
  if (step === "refine" && enabled) {
    return "confidence";
  }
  if (step === "confidence") {
    return "clarity";
  }
  return undefined;
}

export function planningSetupRedirect(input: {
  canUpdate: boolean;
  lineId?: string;
  factoryPresent: boolean;
  linePresent: boolean;
  boardHref: string;
}): string | undefined {
  if (!input.canUpdate || !input.lineId || (input.factoryPresent && !input.linePresent)) {
    return input.boardHref;
  }
  return undefined;
}

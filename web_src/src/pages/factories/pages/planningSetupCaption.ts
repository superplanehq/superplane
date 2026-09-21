import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";

export type PlanningSetupStep = "refine" | "scores";

export function planningSetupPreviewCaption(input: {
  step: PlanningSetupStep;
  enabled: boolean;
  clarity: boolean;
  confidence: boolean;
}): string {
  if (input.step === "refine") {
    return input.enabled
      ? PLANNING_SETTINGS_COPY.wizardPreviewRefineCaption
      : PLANNING_SETTINGS_COPY.wizardPreviewSourceCaption;
  }
  if (input.clarity && input.confidence) {
    return PLANNING_SETTINGS_COPY.wizardPreviewScoresBoth;
  }
  if (input.clarity) {
    return PLANNING_SETTINGS_COPY.wizardPreviewClarityOnly;
  }
  if (input.confidence) {
    return PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOnly;
  }
  return PLANNING_SETTINGS_COPY.wizardPreviewNoScores;
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

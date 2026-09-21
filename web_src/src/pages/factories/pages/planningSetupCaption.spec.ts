import { describe, expect, it } from "bun:test";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { planningSetupPreviewCaption } from "./planningSetupCaption";

describe("planningSetupPreviewCaption", () => {
  it("describes chat refine or the source description", () => {
    expect(planningSetupPreviewCaption({ step: "refine", enabled: true, clarity: true, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewRefineCaption,
    );
    expect(planningSetupPreviewCaption({ step: "refine", enabled: false, clarity: true, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewSourceCaption,
    );
  });

  it("describes which scores appear when Planning is on", () => {
    expect(planningSetupPreviewCaption({ step: "scores", enabled: true, clarity: true, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewScoresBoth,
    );
    expect(planningSetupPreviewCaption({ step: "scores", enabled: true, clarity: true, confidence: false })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewClarityOnly,
    );
    expect(planningSetupPreviewCaption({ step: "scores", enabled: true, clarity: false, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOnly,
    );
    expect(planningSetupPreviewCaption({ step: "scores", enabled: true, clarity: false, confidence: false })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewNoScores,
    );
  });
});

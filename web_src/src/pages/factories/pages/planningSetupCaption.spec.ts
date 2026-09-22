import { describe, expect, it } from "bun:test";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import {
  planningSetupChildStep,
  planningSetupParentStep,
  planningSetupPreviewCaption,
  planningSetupPreviewScene,
} from "./planningSetupCaption";

describe("planningSetupPreviewCaption", () => {
  it("describes chat refine or the source description", () => {
    expect(planningSetupPreviewCaption({ step: "refine", enabled: true, clarity: true, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewRefineCaption,
    );
    expect(planningSetupPreviewCaption({ step: "refine", enabled: false, clarity: true, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewSourceCaption,
    );
  });

  it("describes the clarity check on the Clarity page", () => {
    expect(planningSetupPreviewCaption({ step: "clarity", enabled: true, clarity: true, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewClarityOn,
    );
    expect(planningSetupPreviewCaption({ step: "clarity", enabled: true, clarity: false, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewClarityOff,
    );
  });

  it("describes the confidence estimate on the Confidence page", () => {
    expect(planningSetupPreviewCaption({ step: "confidence", enabled: true, clarity: true, confidence: true })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOn,
    );
    expect(planningSetupPreviewCaption({ step: "confidence", enabled: true, clarity: false, confidence: false })).toBe(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOff,
    );
  });
});

describe("planningSetupPreviewScene", () => {
  it("shows the source task with Start when Planning is off", () => {
    expect(planningSetupPreviewScene({ step: "refine", enabled: false, clarity: true, confidence: true })).toEqual({
      body: "source",
      chat: "question",
      scores: [],
      note: "ready",
      showStart: true,
    });
  });

  it("asks one question and shows the plan without scores on Refine when Planning is on", () => {
    expect(planningSetupPreviewScene({ step: "refine", enabled: true, clarity: true, confidence: true })).toEqual({
      body: "plan",
      chat: "question",
      scores: [],
      note: "ready",
      showStart: false,
    });
  });

  it("offers a split and scores Confidence low on the Confidence page before Clarity is decided", () => {
    const scene = planningSetupPreviewScene({ step: "confidence", enabled: true, clarity: true, confidence: true });
    expect(scene.chat).toBe("confidence");
    expect(scene.scores).toEqual([
      {
        kind: "confidence",
        score: 2,
        summary: PLANNING_SETTINGS_COPY.wizardPreviewConfidenceSummary,
        emphasized: true,
      },
    ]);
    expect(scene.note).toBe("caution");
  });

  it("returns to the planning chat with no scores when the confidence estimate is off", () => {
    const scene = planningSetupPreviewScene({ step: "confidence", enabled: true, clarity: true, confidence: false });
    expect(scene.chat).toBe("question");
    expect(scene.scores).toEqual([]);
    expect(scene.note).toBe("ready");
  });

  it("asks for details and grades Clarity next to a muted Confidence on the Clarity page", () => {
    const scene = planningSetupPreviewScene({ step: "clarity", enabled: true, clarity: true, confidence: true });
    expect(scene.chat).toBe("clarity");
    expect(scene.scores).toEqual([
      { kind: "confidence", score: 4, summary: undefined, emphasized: false },
      {
        kind: "clarity",
        score: 3,
        summary: PLANNING_SETTINGS_COPY.wizardPreviewClaritySummary,
        emphasized: true,
      },
    ]);
    expect(scene.note).toBe("ready");
  });

  it("returns to the planning chat and drops the Clarity row when the clarity check is off", () => {
    const scene = planningSetupPreviewScene({ step: "clarity", enabled: true, clarity: false, confidence: true });
    expect(scene.chat).toBe("question");
    expect(scene.scores.map((score) => score.kind)).toEqual(["confidence"]);
  });
});

describe("planningSetup step links", () => {
  it("walks Refine to Confidence to Clarity when Planning is on", () => {
    expect(planningSetupChildStep("refine", true)).toBe("confidence");
    expect(planningSetupChildStep("confidence", true)).toBe("clarity");
    expect(planningSetupChildStep("clarity", true)).toBeUndefined();
    expect(planningSetupChildStep("refine", false)).toBeUndefined();
  });

  it("walks Clarity to Confidence to Refine", () => {
    expect(planningSetupParentStep("clarity")).toBe("confidence");
    expect(planningSetupParentStep("confidence")).toBe("refine");
    expect(planningSetupParentStep("refine")).toBeUndefined();
  });
});

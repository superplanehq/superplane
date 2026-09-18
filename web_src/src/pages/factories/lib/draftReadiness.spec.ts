import { describe, expect, it } from "bun:test";

import {
  DRAFT_READINESS_NOTES,
  DRAFT_READINESS_SHORT_LABEL,
  draftReadiness,
  liveDraftReadiness,
  startEmphasisForTone,
} from "./draftReadiness";

describe("draftReadiness", () => {
  it("reports analyzing while no score exists and the agent works", () => {
    expect(draftReadiness({ isAnalyzing: true })).toEqual({ tone: "analyzing", ...DRAFT_READINESS_NOTES.analyzing });
  });

  it("reports pending while no score exists and the agent is idle", () => {
    expect(draftReadiness({})).toEqual({ tone: "pending", ...DRAFT_READINESS_NOTES.pending });
  });

  it("blocks when Clarity is 2 or lower", () => {
    expect(draftReadiness({ clarity: 2, confidence: 5 })).toEqual({
      tone: "blocked",
      ...DRAFT_READINESS_NOTES.unclear,
    });
    expect(draftReadiness({ clarity: 0 }).tone).toBe("blocked");
  });

  it("cautions about agent fit when Confidence is 2 or lower and Clarity is fine", () => {
    expect(draftReadiness({ clarity: 5, confidence: 2 })).toEqual({
      tone: "caution",
      ...DRAFT_READINESS_NOTES.agentFit,
    });
    expect(draftReadiness({ confidence: 1 })).toEqual({ tone: "caution", ...DRAFT_READINESS_NOTES.agentFit });
  });

  it("cautions when either score is 3", () => {
    expect(draftReadiness({ clarity: 3, confidence: 5 })).toEqual({
      tone: "caution",
      ...DRAFT_READINESS_NOTES.uncertain,
    });
    expect(draftReadiness({ clarity: 5, confidence: 3 }).tone).toBe("caution");
  });

  it("is ready when every known score is 4 or 5", () => {
    expect(draftReadiness({ clarity: 4, confidence: 5 })).toEqual({ tone: "ready", ...DRAFT_READINESS_NOTES.ready });
    expect(draftReadiness({ confidence: 4 }).tone).toBe("ready");
    expect(draftReadiness({ clarity: 5, isAnalyzing: true }).tone).toBe("ready");
  });

  it("prefers the Clarity block over a low Confidence", () => {
    expect(draftReadiness({ clarity: 1, confidence: 1 }).tone).toBe("blocked");
  });
});

describe("liveDraftReadiness", () => {
  it("says analyzing while the agent works, even with older scores", () => {
    expect(liveDraftReadiness({ clarity: 5, confidence: 5, isAnalyzing: true }).tone).toBe("analyzing");
  });

  it("falls back to the score verdict when the agent is idle", () => {
    expect(liveDraftReadiness({ clarity: 5, confidence: 5 }).tone).toBe("ready");
  });
});

describe("startEmphasisForTone", () => {
  it("fills Start only when the verdict says go", () => {
    expect(startEmphasisForTone("ready")).toBe("filled");
    expect(startEmphasisForTone("pending")).toBe("filled");
    expect(startEmphasisForTone("caution")).toBe("outline");
    expect(startEmphasisForTone("blocked")).toBe("outline");
    expect(startEmphasisForTone("analyzing")).toBe("outline");
  });
});

describe("DRAFT_READINESS_SHORT_LABEL", () => {
  it("keeps every card label to two words or fewer", () => {
    for (const label of Object.values(DRAFT_READINESS_SHORT_LABEL)) {
      expect(label.split(" ").length).toBeLessThanOrEqual(2);
    }
  });
});

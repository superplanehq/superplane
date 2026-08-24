import { describe, expect, it } from "vitest";

import { clampConfidenceScore, confidenceBandForScore, confidenceCheckLevel } from "./confidenceScore";

describe("confidenceScore", () => {
  it("clamps scores to 0 through 5", () => {
    expect(clampConfidenceScore(-1)).toBe(0);
    expect(clampConfidenceScore(5.4)).toBe(5);
    expect(clampConfidenceScore(3.6)).toBe(4);
  });

  it("bands and levels a 0 to 5 score", () => {
    expect(confidenceBandForScore(4)).toBe("High");
    expect(confidenceBandForScore(3)).toBe("Medium");
    expect(confidenceBandForScore(2)).toBe("Low");
    expect(confidenceCheckLevel(4)).toBe("positive");
    expect(confidenceCheckLevel(3)).toBe("neutral");
    expect(confidenceCheckLevel(2)).toBe("caution");
  });
});

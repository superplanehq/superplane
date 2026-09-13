import { describe, expect, it } from "vitest";

import {
  analysisFinishedStatus,
  analysisFirstResultDelivered,
  hasAnalysisPlan,
  hasAnalysisScore,
} from "./analysisOutcome";

describe("analysisFirstResultDelivered", () => {
  it("is true when a score and spec.md exist", () => {
    expect(
      analysisFirstResultDelivered({
        checks: [{ name: "Confidence score", key: "confidence", score: 4 }],
        artifacts: [{ data: { name: "spec.md", body: "# Add breed\n\n## Executive summary\n\nAdd breed.\n" } }],
      }),
    ).toBe(true);
  });

  it("is false when the score is missing", () => {
    expect(
      analysisFirstResultDelivered({
        checks: [],
        artifacts: [{ data: { name: "spec.md", body: "# Add breed\n" } }],
      }),
    ).toBe(false);
  });

  it("is false when the plan is missing", () => {
    expect(
      analysisFirstResultDelivered({
        checks: [{ name: "Confidence score", score: 3 }],
        artifacts: [],
      }),
    ).toBe(false);
  });

  it("accepts a score of 0", () => {
    expect(hasAnalysisScore([{ key: "confidence", score: 0 }])).toBe(true);
  });

  it("accepts intent.md as the plan", () => {
    expect(hasAnalysisPlan([{ data: { title: "intent.md", body: "A retry loop." } }])).toBe(true);
  });
});

describe("analysisFinishedStatus", () => {
  it("keeps a cancelled or failed stop as passed after the first result", () => {
    expect(analysisFinishedStatus("cancelled", true)).toBe("passed");
    expect(analysisFinishedStatus("failed", true)).toBe("passed");
  });

  it("leaves a true failure in place when no first result exists", () => {
    expect(analysisFinishedStatus("failed", false)).toBe("failed");
    expect(analysisFinishedStatus("cancelled", false)).toBe("cancelled");
  });

  it("does not change a running analysis", () => {
    expect(analysisFinishedStatus("running", true)).toBe("running");
  });
});

import { describe, expect, it } from "vitest";

import {
  boardCardLoadsConfidenceChecks,
  clampConfidenceScore,
  confidenceBandForScore,
  confidenceCheckLevel,
  confidenceScoreFromChecks,
  confidenceScoreFromPercent,
  confidenceSuitabilityAnalysis,
  confidenceSuitabilitySummary,
} from "./confidenceScore";
import { workOrderCheckStatus } from "./workOrderChecks";

describe("confidenceScore", () => {
  it("loads board confidence checks only for draft cards", () => {
    expect(boardCardLoadsConfidenceChecks("draft")).toBe(true);
    expect(boardCardLoadsConfidenceChecks("waiting")).toBe(false);
    expect(boardCardLoadsConfidenceChecks("running")).toBe(false);
    expect(boardCardLoadsConfidenceChecks("failed")).toBe(false);
    expect(boardCardLoadsConfidenceChecks("completed")).toBe(false);
  });

  it("reads the Confidence score check and ignores other checks", () => {
    expect(
      confidenceScoreFromChecks([
        { name: "Code quality", score: 82 },
        { name: "Confidence score", score: 4.2 },
      ]),
    ).toBe(4);
    expect(confidenceScoreFromChecks([{ name: "Code quality", score: 82 }])).toBeUndefined();
    expect(confidenceScoreFromChecks([])).toBeUndefined();
  });

  it("clamps scores to 0 through 5", () => {
    expect(clampConfidenceScore(-1)).toBe(0);
    expect(clampConfidenceScore(5.4)).toBe(5);
    expect(clampConfidenceScore(3.6)).toBe(4);
  });

  it("turns an intake percentage into a 0 to 5 score", () => {
    expect(confidenceScoreFromPercent(94)).toBe(5);
    expect(confidenceScoreFromPercent(58)).toBe(3);
    expect(confidenceScoreFromPercent(44)).toBe(2);
    expect(confidenceScoreFromPercent(12)).toBe(1);
    expect(confidenceScoreFromPercent(0)).toBe(0);
  });

  it("bands and levels a 0 to 5 score", () => {
    expect(confidenceBandForScore(4)).toBe("High");
    expect(confidenceBandForScore(3)).toBe("Medium");
    expect(confidenceBandForScore(2)).toBe("Low");
    expect(confidenceCheckLevel(4)).toBe("positive");
    expect(confidenceCheckLevel(3)).toBe("neutral");
    expect(confidenceCheckLevel(2)).toBe("caution");
  });

  it("labels a confidence check High Medium or Low, not Healthy", () => {
    expect(workOrderCheckStatus({ name: "Confidence score", score: 5, level: "positive" }).label).toBe("High");
    expect(workOrderCheckStatus({ name: "Confidence score", score: 3, level: "neutral" }).label).toBe("Medium");
    expect(workOrderCheckStatus({ name: "Risk score", score: 65, level: "caution" }).label).toBe("Needs attention");
  });

  it("summarizes how suitable the issue is for an agent", () => {
    expect(confidenceSuitabilitySummary("High")).toBe("This issue is a good fit for an agent on this factory line.");
    expect(confidenceSuitabilitySummary("Medium")).toBe("This issue is a mixed fit for an agent on this factory line.");
    expect(confidenceSuitabilitySummary("Low")).toBe("This issue is a poor fit for an agent on this factory line.");
  });

  it("explains that the automation scored the source issue for agent fit", () => {
    expect(confidenceSuitabilityAnalysis({ source: "GitHub", reasons: ["The dispatcher is mapped."] })).toBe(
      [
        "The automation read this GitHub issue. It scored how suitable the work is for an agent on this factory line.",
        "",
        "### Why this score",
        "- The dispatcher is mapped.",
      ].join("\n"),
    );
    expect(confidenceSuitabilityAnalysis({ source: "Sentry" })).toBe(
      "The automation read this Sentry issue. It scored how suitable the work is for an agent on this factory line.",
    );
  });
});

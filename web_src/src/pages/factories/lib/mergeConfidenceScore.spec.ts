import { describe, expect, it } from "bun:test";

import { consoleCheckList, mergeConfidencePanelTone, mergeConfidenceScore } from "./mergeConfidenceScore";
import type { WorkOrderCheckPresentation } from "./workOrderChecks";

function check(
  overrides: Partial<WorkOrderCheckPresentation> & Pick<WorkOrderCheckPresentation, "id" | "key" | "name">,
): WorkOrderCheckPresentation {
  return {
    score: 5,
    maxScore: 5,
    level: "positive",
    ...overrides,
  };
}

describe("mergeConfidenceScore", () => {
  it("uses the weakest metric so a caution stays on the headline", () => {
    const score = mergeConfidenceScore([
      check({ id: "risk", key: "risk-review", name: "Risk score", score: 3, level: "caution" }),
      check({ id: "drift", key: "drift-review", name: "Drift", score: 2 }),
      check({ id: "performance", key: "performance-review", name: "Performance", score: 5 }),
      check({ id: "security", key: "security-review", name: "Security", score: 5 }),
    ]);

    expect(score).toMatchObject({ name: "Merge confidence", score: 3, maxScore: 5, level: "caution" });
    expect(mergeConfidencePanelTone(score)).toContain("--color-orange-50");
  });

  it("treats a low risk score as high merge confidence", () => {
    const score = mergeConfidenceScore([
      check({ id: "risk", key: "risk-review", name: "Risk score", score: 1 }),
      check({ id: "drift", key: "drift-review", name: "Drift", score: 1 }),
    ]);

    expect(score.score).toBe(5);
    expect(score.level).toBe("positive");
  });
});

describe("consoleCheckList", () => {
  it("drops planning confidence and keeps only the merge confidence score", () => {
    const items = consoleCheckList([
      check({ id: "confidence", key: "confidence", name: "Confidence score", score: 2, level: "caution" }),
      check({ id: "drift", key: "drift-review", name: "Drift", score: 2 }),
      check({ id: "risk", key: "risk-review", name: "Risk score", score: 3, level: "caution" }),
      check({ id: "coverage", key: "code-coverage", name: "Code quality", score: 80, maxScore: 100 }),
    ]);

    expect(items).toMatchObject({
      check: { score: 3, level: "caution" },
      metrics: [{ name: "Risk score" }, { name: "Drift" }],
    });
  });
});

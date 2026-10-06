import { describe, expect, it } from "bun:test";

import { planningReviewFromChecks, planningReviewHeadline } from "./planningReviewScore";
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

describe("planningReviewFromChecks", () => {
  it("returns null until all three sub-parameters exist", () => {
    expect(
      planningReviewFromChecks([
        check({ id: "clarity", key: "clarity", name: "Clarity score", score: 5 }),
        check({ id: "confidence", key: "confidence", name: "Confidence score", score: 4 }),
      ]),
    ).toBeNull();
  });

  it("uses the weakest sub-parameter as the Confidence headline", () => {
    const review = planningReviewFromChecks([
      check({ id: "clarity", key: "clarity", name: "Clarity score", score: 5, summary: "Defined." }),
      check({
        id: "complexity",
        key: "complexity",
        name: "Complexity",
        score: 3,
        level: "caution",
        summary: "The change is large.",
      }),
      check({ id: "risk", key: "risk", name: "Risk", score: 5, summary: "Easy to undo." }),
    ]);

    expect(review?.headline).toMatchObject({
      name: "Confidence score",
      score: 3,
      level: "caution",
      summary: "The change is large.",
    });
    expect(review?.metrics.map((metric) => metric.name)).toEqual(["Clarity", "Complexity", "Risk"]);
  });
});

describe("planningReviewHeadline", () => {
  it("returns undefined for an empty list", () => {
    expect(planningReviewHeadline([])).toBeUndefined();
  });
});

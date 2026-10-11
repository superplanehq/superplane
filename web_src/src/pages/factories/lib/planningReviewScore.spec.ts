import { describe, expect, it } from "bun:test";

import {
  planningReviewAtMax,
  planningReviewFromChecks,
  planningReviewHeadline,
  planningReviewLevel,
} from "./planningReviewScore";
import type { WorkOrderCheckPresentation } from "./workOrderChecks";

function check(
  overrides: Partial<WorkOrderCheckPresentation> & Pick<WorkOrderCheckPresentation, "id" | "key" | "name">,
): WorkOrderCheckPresentation {
  return {
    score: 3,
    maxScore: 3,
    level: "positive",
    ...overrides,
  };
}

describe("planningReviewFromChecks", () => {
  it("returns null until all three sub-parameters exist", () => {
    expect(
      planningReviewFromChecks([
        check({ id: "clarity", key: "clarity", name: "Clarity score", score: 3 }),
        check({ id: "confidence", key: "confidence", name: "Confidence score", score: 2 }),
      ]),
    ).toBeNull();
  });

  it("ignores checks stored on the old 1 through 5 scale", () => {
    expect(
      planningReviewFromChecks([
        check({ id: "clarity", key: "clarity", name: "Clarity score", score: 5, maxScore: 5 }),
        check({ id: "complexity", key: "complexity", name: "Complexity", score: 4, maxScore: 5 }),
        check({ id: "verifiability", key: "verifiability", name: "Verifiability", score: 5, maxScore: 5 }),
      ]),
    ).toBeNull();
  });

  it("uses the weakest sub-parameter as the Confidence headline", () => {
    const review = planningReviewFromChecks([
      check({ id: "clarity", key: "clarity", name: "Clarity score", score: 3, summary: "Defined." }),
      check({
        id: "complexity",
        key: "complexity",
        name: "Complexity",
        score: 2,
        level: "caution",
        summary: "The change is large.",
      }),
      check({
        id: "verifiability",
        key: "verifiability",
        name: "Verifiability",
        score: 3,
        summary: "Existing tests cover the change.",
      }),
    ]);

    expect(review?.headline).toMatchObject({
      name: "Confidence score",
      score: 2,
      maxScore: 3,
      level: "caution",
      summary: "The change is large.",
    });
    expect(review?.metrics.map((metric) => metric.name)).toEqual(["Clarity", "Complexity", "Verifiability"]);
  });
});

describe("planningReviewHeadline", () => {
  it("returns undefined for an empty list", () => {
    expect(planningReviewHeadline([])).toBeUndefined();
  });
});

describe("planningReviewLevel", () => {
  it("matches the stored review bands, not the headline confidence bands", () => {
    expect(planningReviewLevel(3)).toBe("positive");
    expect(planningReviewLevel(2)).toBe("caution");
    expect(planningReviewLevel(1)).toBe("critical");
  });
});

describe("planningReviewAtMax", () => {
  it("is true only when every sub-parameter sits at 3", () => {
    expect(planningReviewAtMax([{ score: 3 }, { score: 3 }, { score: 3 }])).toBe(true);
    expect(planningReviewAtMax([{ score: 3 }, { score: 2 }, { score: 3 }])).toBe(false);
    expect(planningReviewAtMax([])).toBe(false);
  });
});

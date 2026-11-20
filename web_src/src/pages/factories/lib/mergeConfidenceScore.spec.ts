import { describe, expect, it } from "bun:test";

import {
  checksForRepository,
  consoleCheckList,
  mergeConfidenceCheckLevel,
  mergeConfidenceHeadline,
  mergeConfidencePanelTone,
  mergeConfidenceScore,
  splitMergeChecksByPullRequest,
} from "./mergeConfidenceScore";
import {
  formatCheckScore,
  presentWorkOrderCheck,
  workOrderCheckStatus,
  type WorkOrderCheckPresentation,
} from "./workOrderChecks";

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
  it.each([
    [1, 5, "Pass"],
    [2, 4, "Pass"],
    [3, 3, "Needs review"],
    [4, 2, "Fail"],
    [5, 1, "Fail"],
  ] as const)(
    "shows legacy severity %i and new Correctness %i with the same verdict (%s)",
    (legacy, displayed, verdict) => {
      for (const [key, score] of [
        ["drift-review", legacy],
        ["correctness-review", displayed],
      ] as const) {
        const correctness = presentWorkOrderCheck({ key, name: "Correctness", score, maxScore: 5 });
        correctness.level = mergeConfidenceCheckLevel(key, score);
        const metrics = [correctness, check({ id: "security", key: "security-review", name: "Security" })];

        expect(formatCheckScore(correctness).value).toBe(String(displayed));
        expect(workOrderCheckStatus(correctness).label).toBe(verdict);
        expect(mergeConfidenceScore(metrics).score).toBe(displayed);
        expect(mergeConfidenceHeadline(metrics)?.score).toBe(displayed);
      }
    },
  );

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

describe("mergeConfidenceCheckLevel", () => {
  it("uses the server bands for each check", () => {
    expect(mergeConfidenceCheckLevel("risk-review", 1)).toBe("positive");
    expect(mergeConfidenceCheckLevel("risk-review", 3)).toBe("caution");
    expect(mergeConfidenceCheckLevel("risk-review", 4)).toBe("critical");
    expect(mergeConfidenceCheckLevel("drift-review", 4)).toBe("critical");
    expect(mergeConfidenceCheckLevel("performance-review", 5)).toBe("positive");
    expect(mergeConfidenceCheckLevel("performance-review", 3)).toBe("caution");
    expect(mergeConfidenceCheckLevel("performance-review", 2)).toBe("critical");
    expect(mergeConfidenceCheckLevel("security-review", 3)).toBe("caution");
    expect(mergeConfidenceCheckLevel("reversibility-review", 5)).toBe("positive");
    expect(mergeConfidenceCheckLevel("api-latency-review", 4)).toBe("critical");
  });
});

describe("mergeConfidenceHeadline", () => {
  it("keeps each check for the card list", () => {
    const headline = mergeConfidenceHeadline([
      { key: "security-review", name: "Security", score: 5, maxScore: 5 },
      { key: "risk-review", name: "Blast radius", score: 1, maxScore: 5 },
      { key: "confidence", name: "Confidence score", score: 2, maxScore: 5 },
    ]);

    expect(headline).toMatchObject({
      score: 5,
      checks: [
        { key: "security-review", name: "Security" },
        { key: "risk-review", name: "Blast radius" },
      ],
    });
  });

  it("lists Correctness once with its weakest score across repositories using either scale", () => {
    const headline = mergeConfidenceHeadline([
      { key: "correctness-review", name: "Correctness", score: 5, maxScore: 5 },
      { key: "drift-review", name: "Drift", score: 4, maxScore: 5 },
    ]);

    expect(headline).toMatchObject({
      score: 2,
      checks: [{ key: "drift-review", score: 4 }],
    });
    expect(headline?.checks).toHaveLength(1);
  });

  it("drops checks that omit a score and still scores the rest", () => {
    const headline = mergeConfidenceHeadline([
      { key: "security-review", name: "Security" },
      { key: "performance-review", name: "Performance", score: 4, maxScore: 0 },
      { key: "risk-review", name: "Blast radius", score: 1, maxScore: 5 },
    ]);

    expect(headline).toMatchObject({
      score: 5,
      checks: [{ key: "risk-review", name: "Blast radius", score: 1, maxScore: 5 }],
    });
  });
});

describe("checksForRepository", () => {
  it("matches the repository without case and keeps checks of the whole task", () => {
    const checks = [
      { id: "api", repository: "Acme/API" },
      { id: "docs", repository: "acme/docs" },
      { id: "task", repository: undefined },
    ];

    expect(checksForRepository(checks, "acme/api").map((check) => check.id)).toEqual(["api", "task"]);
  });
});

describe("splitMergeChecksByPullRequest", () => {
  const api = check({ id: "api", key: "risk-review", name: "Blast radius", repository: "Acme/API" });
  const web = check({ id: "web", key: "security-review", name: "Security", repository: "acme/web" });
  const task = check({ id: "task", key: "drift-review", name: "Drift" });
  const other = check({ id: "other", key: "risk-review", name: "Blast radius", repository: "acme/other" });
  const ids = (checks: WorkOrderCheckPresentation[] | undefined) => (checks ?? []).map((entry) => entry.id);

  it("gives each pull request the checks of its repository", () => {
    const split = splitMergeChecksByPullRequest([api, web, other], ["acme/api", "acme/web"]);

    expect(ids(split.byRepository.get("acme/api"))).toEqual(["api"]);
    expect(ids(split.byRepository.get("acme/web"))).toEqual(["web"]);
    expect(ids(split.unattached)).toEqual(["other"]);
  });

  it("keeps a check without a repository off the rows when several repositories have pull requests", () => {
    const split = splitMergeChecksByPullRequest([api, task], ["acme/api", "acme/web"]);

    expect(ids(split.unattached)).toEqual(["task"]);
  });

  it("attaches a check without a repository to the only pull request repository", () => {
    const split = splitMergeChecksByPullRequest([api, task], ["acme/api", "ACME/api"]);

    expect(ids(split.byRepository.get("acme/api"))).toEqual(["task", "api"]);
    expect(split.unattached).toEqual([]);
  });

  it("leaves every check unattached without pull requests", () => {
    expect(ids(splitMergeChecksByPullRequest([api, task], []).unattached)).toEqual(["task", "api"]);
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
      metrics: [{ name: "Drift" }, { name: "Risk score" }],
    });
  });
});

import { describe, expect, it } from "bun:test";

import type { FactoriesFactoryPullRequest } from "@/api-client";

import {
  selectWorkOrderCardPullRequest,
  visibleWorkOrderCardAttentionReasons,
  workOrderCardPullRequestAriaLabel,
  workOrderCardPullRequestShowsMergeable,
  workOrderCardPullRequestVisibleLabel,
} from "./workOrderCardPullRequest";

function pr(overrides: Partial<FactoriesFactoryPullRequest> = {}): FactoriesFactoryPullRequest {
  return {
    id: "pr-1",
    workOrderId: "wo-1",
    number: "2323",
    url: "https://github.com/acme/payments/pull/2323",
    title: "Ship idempotent refund retries",
    state: "STATE_OPEN",
    ...overrides,
  };
}

describe("selectWorkOrderCardPullRequest", () => {
  it("returns null when the task has no pull request", () => {
    expect(selectWorkOrderCardPullRequest([], "wo-1")).toBeNull();
    expect(selectWorkOrderCardPullRequest(undefined, "wo-1")).toBeNull();
  });

  it("keeps pull requests that belong to this task", () => {
    const selected = selectWorkOrderCardPullRequest(
      [pr({ workOrderId: "wo-other", number: "9" }), pr({ id: "pr-mine" })],
      "wo-1",
    );
    expect(selected?.pullRequest.id).toBe("pr-mine");
    expect(selected?.extraCount).toBe(0);
  });

  it("prefers an open pull request over a merged one", () => {
    const selected = selectWorkOrderCardPullRequest(
      [pr({ id: "merged", number: "99", state: "STATE_MERGED" }), pr({ id: "open", number: "12" })],
      "wo-1",
    );
    expect(selected?.pullRequest.id).toBe("open");
    expect(selected?.extraCount).toBe(1);
  });

  it("picks the higher number when both pull requests are open", () => {
    const selected = selectWorkOrderCardPullRequest(
      [pr({ id: "older", number: "10" }), pr({ id: "newer", number: "2323" })],
      "wo-1",
    );
    expect(selected?.pullRequest.id).toBe("newer");
    expect(selected?.extraCount).toBe(1);
  });
});

describe("workOrderCardPullRequestVisibleLabel", () => {
  it("names an open pull request as Review plus the number", () => {
    expect(workOrderCardPullRequestVisibleLabel(pr())).toBe("Review #2323");
  });

  it("names draft, merged, and closed states", () => {
    expect(workOrderCardPullRequestVisibleLabel(pr({ state: "STATE_DRAFT" }))).toBe("Draft #2323");
    expect(workOrderCardPullRequestVisibleLabel(pr({ state: "STATE_MERGED" }))).toBe("Merged #2323");
    expect(workOrderCardPullRequestVisibleLabel(pr({ state: "STATE_CLOSED" }))).toBe("Closed #2323");
  });

  it("appends a count when more pull requests are attached", () => {
    expect(workOrderCardPullRequestVisibleLabel(pr(), 1)).toBe("Review #2323 +1");
  });
});

describe("workOrderCardPullRequestShowsMergeable", () => {
  const open = pr({ mergeable: true });

  it("hides the badge when any scored merge confidence metric is critical", () => {
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "risk-review", score: 4 }])).toBe(false);
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "drift-review", score: 5 }])).toBe(false);
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "performance-review", score: 2 }])).toBe(false);
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "security-review", score: 1 }])).toBe(false);
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "reversibility-review", score: 2 }])).toBe(false);
    expect(
      workOrderCardPullRequestShowsMergeable(open, [
        { key: "security-review", score: 5 },
        { key: "risk-review", score: 4 },
      ]),
    ).toBe(false);
  });

  it("keeps the badge when scored merge confidence metrics are caution or better", () => {
    expect(
      workOrderCardPullRequestShowsMergeable(open, [
        { key: "risk-review", score: 3 },
        { key: "drift-review", score: 3 },
        { key: "performance-review", score: 3 },
        { key: "security-review", score: 5 },
        { key: "reversibility-review", score: 4 },
      ]),
    ).toBe(true);
  });

  it("keeps the badge when no scored merge confidence metric exists", () => {
    expect(workOrderCardPullRequestShowsMergeable(open, undefined)).toBe(true);
    expect(workOrderCardPullRequestShowsMergeable(open, [])).toBe(true);
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "risk-review" }])).toBe(true);
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "planning-review", score: 1 }])).toBe(true);
    expect(workOrderCardPullRequestShowsMergeable(open, [{ key: "api-latency-review", score: 5 }])).toBe(true);
  });

  it("stays hidden when the pull request is not open and mergeable", () => {
    expect(workOrderCardPullRequestShowsMergeable(pr(), [{ key: "risk-review", score: 1 }])).toBe(false);
    expect(workOrderCardPullRequestShowsMergeable(undefined, [])).toBe(false);
  });
});

describe("visibleWorkOrderCardAttentionReasons", () => {
  it("hides Status checks passed when the pull request is mergeable and the flag is on", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(
        ["approval", "checksPassed"],
        {
          pullRequest: pr({ mergeable: true }),
          extraCount: 0,
        },
        true,
      ),
    ).toEqual([]);
  });

  it("keeps Status checks passed when the pull request is mergeable but the flag is off", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(
        ["approval", "checksPassed"],
        {
          pullRequest: pr({ mergeable: true }),
          extraCount: 0,
        },
        false,
      ),
    ).toEqual(["checksPassed"]);
  });

  it("keeps Status checks passed when the pull request is mergeable and the flag is undefined", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(["approval", "checksPassed"], {
        pullRequest: pr({ mergeable: true }),
        extraCount: 0,
      }),
    ).toEqual(["checksPassed"]);
  });

  it("keeps Status checks passed when a merge confidence check is critical", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(
        ["checksPassed"],
        { pullRequest: pr({ mergeable: true }), extraCount: 0 },
        true,
        [{ key: "security-review", score: 2 }],
      ),
    ).toEqual(["checksPassed"]);
  });

  it("hides Status checks passed when merge confidence checks are only caution", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(
        ["checksPassed"],
        { pullRequest: pr({ mergeable: true }), extraCount: 0 },
        true,
        [{ key: "risk-review", score: 3 }],
      ),
    ).toEqual([]);
  });

  it("hides Waiting for user review when the shown pull request is open", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(["approval", "checksPassed"], { pullRequest: pr(), extraCount: 0 }),
    ).toEqual(["checksPassed"]);
  });

  it("hides Waiting for user review when the shown pull request is merged", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(["approval", "failed"], {
        pullRequest: pr({ state: "STATE_MERGED" }),
        extraCount: 0,
      }),
    ).toEqual(["failed"]);
  });

  it("keeps review when the pull request is not open", () => {
    expect(
      visibleWorkOrderCardAttentionReasons(["approval", "failed"], {
        pullRequest: pr({ state: "STATE_CLOSED" }),
        extraCount: 0,
      }),
    ).toEqual(["approval", "failed"]);
  });

  it("does not invent an idle attention reason next to an open pull request", () => {
    expect(visibleWorkOrderCardAttentionReasons([], { pullRequest: pr(), extraCount: 0 })).toEqual([]);
  });
});

describe("workOrderCardPullRequestAriaLabel", () => {
  it("says Review pull request and the number", () => {
    expect(workOrderCardPullRequestAriaLabel(pr())).toBe("Review pull request #2323.");
  });

  it("names extra pull requests in the accessible label", () => {
    expect(workOrderCardPullRequestAriaLabel(pr(), 1)).toBe("Review pull request #2323. 1 more pull request.");
    expect(workOrderCardPullRequestAriaLabel(pr(), 2)).toBe("Review pull request #2323. 2 more pull requests.");
  });
});

import { describe, expect, it } from "bun:test";

import { defaultMergeMethod, pullRequestReviewNote } from "./splitRunPullRequestReview";

describe("pullRequestReviewNote", () => {
  it("recognizes a note whose call to action opens a GitHub pull request", () => {
    expect(
      pullRequestReviewNote({
        headline: "Waiting for user review",
        cta: { label: "Review PR #6812", href: "https://github.com/acme/payments/pull/6812" },
      }),
    ).toEqual({ href: "https://github.com/acme/payments/pull/6812", number: 6812 });
  });

  it("accepts a pull request link with a trailing path or query", () => {
    expect(
      pullRequestReviewNote({
        headline: "Review",
        cta: { label: "Review", href: "https://github.com/acme/payments/pull/12/files?diff=split" },
      }),
    ).toEqual({ href: "https://github.com/acme/payments/pull/12/files?diff=split", number: 12 });
  });

  it("ignores notes without a link", () => {
    expect(pullRequestReviewNote({ headline: "Implement did not pass", cta: { label: "Debug" } })).toBeUndefined();
    expect(pullRequestReviewNote({ headline: "No pull request" })).toBeUndefined();
  });

  it("ignores links that are not pull requests", () => {
    expect(
      pullRequestReviewNote({
        headline: "Review",
        cta: { label: "Open issue", href: "https://github.com/acme/payments/issues/6812" },
      }),
    ).toBeUndefined();
    expect(
      pullRequestReviewNote({
        headline: "Debug",
        cta: { label: "Debug", href: "/org/workspaces/key/apps/app-1/runs/run-1" },
      }),
    ).toBeUndefined();
  });
});

describe("defaultMergeMethod", () => {
  it("prefers squash, then a merge commit, then rebase", () => {
    expect(defaultMergeMethod(["MERGE_METHOD_REBASE", "MERGE_METHOD_SQUASH"])).toBe("MERGE_METHOD_SQUASH");
    expect(defaultMergeMethod(["MERGE_METHOD_REBASE", "MERGE_METHOD_MERGE"])).toBe("MERGE_METHOD_MERGE");
    expect(defaultMergeMethod(["MERGE_METHOD_REBASE"])).toBe("MERGE_METHOD_REBASE");
  });
});

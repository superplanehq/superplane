import type { CanvasesCanvasRun } from "@/api-client";
import { describe, expect, it } from "bun:test";

import {
  mergeConfidenceCanvases,
  mergeConfidenceRunsForPullRequests,
  pullRequestIdentityFromRootEvent,
} from "./mergeConfidenceRuns";

function scoreRun(overrides: {
  id: string;
  number?: number;
  repository?: string;
  url?: string;
  createdAt?: string;
  wrapped?: boolean;
}): CanvasesCanvasRun {
  const payload = {
    number: overrides.number,
    pull_request: {
      number: overrides.number,
      html_url: overrides.url,
      base: { repo: { full_name: overrides.repository } },
    },
    repository: { full_name: overrides.repository },
  };
  return {
    id: overrides.id,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: overrides.createdAt ?? "2026-08-26T11:00:00Z",
    rootEvent: {
      data: overrides.wrapped === false ? payload : { type: "github.pullRequest", data: payload },
    },
  };
}

const canvas = { id: "app-merge", name: "Merge confidence" };

describe("mergeConfidenceCanvases", () => {
  it("keeps Verify copies and skips other names", () => {
    expect(
      mergeConfidenceCanvases([
        { id: "app-merge", name: "Merge confidence", columnKey: "verify" },
        { id: "app-merge-2", name: "Merge confidence (2)", columnKey: "verify" },
        { id: "app-risk", name: "Risk score", columnKey: "verify" },
        { id: "app-risk-2", name: "Risk score (2)", columnKey: "verify" },
        { id: "app-payments", name: "Merge confidence payments", columnKey: "verify" },
        { id: "app-done", name: "Merge confidence", columnKey: "done" },
      ]).map((entry) => entry.id),
    ).toEqual(["app-merge", "app-merge-2", "app-risk", "app-risk-2"]);
  });
});

describe("mergeConfidenceRunsForPullRequests", () => {
  const pullRequests = [{ number: "12", repository: "acme/app", url: "https://github.com/acme/app/pull/12" }];

  it("matches the pull request number and repository from the trigger envelope", () => {
    const runs = mergeConfidenceRunsForPullRequests(
      canvas,
      [
        scoreRun({ id: "run-other", number: 12, repository: "acme/other" }),
        scoreRun({ id: "run-match", number: 12, repository: "acme/app", createdAt: "2026-08-26T12:00:00Z" }),
        scoreRun({ id: "run-older", number: 12, repository: "acme/app", createdAt: "2026-08-26T10:00:00Z" }),
      ],
      pullRequests,
    );

    expect(runs.map((entry) => entry.run.id)).toEqual(["run-older", "run-match"]);
    expect(runs[0]?.canvasName).toBe("Merge confidence");
  });

  it("matches a pull request URL when the number is absent", () => {
    const run = scoreRun({
      id: "run-url",
      url: "https://github.com/acme/app/pull/12/",
    });
    run.rootEvent = { data: { data: { pull_request: { html_url: "https://github.com/acme/app/pull/12/" } } } };

    expect(mergeConfidenceRunsForPullRequests(canvas, [run], [{ url: "https://github.com/acme/app/pull/12" }])).toEqual(
      [expect.objectContaining({ canvasId: "app-merge" })],
    );
  });

  it("reads an unwrapped webhook the same way as the event envelope", () => {
    expect(
      pullRequestIdentityFromRootEvent(scoreRun({ id: "run-raw", number: 7, repository: "acme/app", wrapped: false })),
    ).toMatchObject({ number: "7", repository: "acme/app" });
  });
});

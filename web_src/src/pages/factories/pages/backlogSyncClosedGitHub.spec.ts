import { describe, expect, it } from "vitest";

import {
  BACKLOG_SYNC_GITHUB_COPY,
  canSyncClosedGitHubIssues,
  hasGitHubIssuesIntake,
  syncClosedGitHubToast,
} from "./backlogSyncClosedGitHub";

describe("hasGitHubIssuesIntake", () => {
  it("is true when a GitHub issues intake exists", () => {
    expect(hasGitHubIssuesIntake([{ source: "SOURCE_GITHUB_ISSUES" }])).toBe(true);
  });

  it("is false when GitHub issues are not an intake source", () => {
    expect(hasGitHubIssuesIntake([{ source: "SOURCE_SENTRY_EXCEPTIONS" }])).toBe(false);
    expect(hasGitHubIssuesIntake([])).toBe(false);
    expect(hasGitHubIssuesIntake(undefined)).toBe(false);
  });
});

describe("canSyncClosedGitHubIssues", () => {
  it("is true when a GitHub issues intake exists and the user can update work orders", () => {
    expect(canSyncClosedGitHubIssues([{ source: "SOURCE_GITHUB_ISSUES" }], true)).toBe(true);
  });

  it("is false when the user cannot update work orders", () => {
    expect(canSyncClosedGitHubIssues([{ source: "SOURCE_GITHUB_ISSUES" }], false)).toBe(false);
  });
});

describe("syncClosedGitHubToast", () => {
  it("reports no tasks when nothing closed", () => {
    expect(syncClosedGitHubToast({ closedCount: 0, failedCount: 0 })).toEqual({
      kind: "info",
      message: BACKLOG_SYNC_GITHUB_COPY.none,
    });
  });

  it("reports the closed count", () => {
    expect(syncClosedGitHubToast({ closedCount: 1, failedCount: 0 })).toEqual({
      kind: "success",
      message: BACKLOG_SYNC_GITHUB_COPY.closedOne,
    });
    expect(syncClosedGitHubToast({ closedCount: 3, failedCount: 0 })).toEqual({
      kind: "success",
      message: "Closed 3 tasks.",
    });
  });

  it("reports a GitHub failure when no task closed", () => {
    expect(syncClosedGitHubToast({ closedCount: 0, failedCount: 2 })).toEqual({
      kind: "error",
      message: BACKLOG_SYNC_GITHUB_COPY.failed,
    });
  });

  it("joins closed and failed counts", () => {
    expect(syncClosedGitHubToast({ closedCount: 2, failedCount: 1 })).toEqual({
      kind: "error",
      message: "Closed 2 tasks. SuperPlane could not check 1 GitHub issue.",
    });
  });
});

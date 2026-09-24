import { describe, expect, it } from "bun:test";

import { BACKLOG_REFRESH_COPY, backlogRefreshToast, canRefreshBacklog, hasRefreshableIntake } from "./backlogRefresh";

describe("hasRefreshableIntake", () => {
  it("accepts intake sources that support item lookup", () => {
    expect(hasRefreshableIntake([{ source: "SOURCE_GITHUB_ISSUES" }])).toBe(true);
    expect(hasRefreshableIntake([{ source: "SOURCE_PRODUCTIVE_TASKS" }])).toBe(true);
  });

  it("rejects intake sources that cannot look up items", () => {
    expect(hasRefreshableIntake([{ source: "SOURCE_SENTRY_EXCEPTIONS" }])).toBe(false);
    expect(hasRefreshableIntake([])).toBe(false);
    expect(hasRefreshableIntake(undefined)).toBe(false);
  });
});

describe("canRefreshBacklog", () => {
  it("requires a refreshable intake and work-order update permission", () => {
    expect(canRefreshBacklog([{ source: "SOURCE_GITHUB_ISSUES" }], true)).toBe(true);
    expect(canRefreshBacklog([{ source: "SOURCE_GITHUB_ISSUES" }], false)).toBe(false);
  });
});

describe("backlogRefreshToast", () => {
  it("reports an up-to-date backlog", () => {
    expect(backlogRefreshToast({ archivedCount: 0, failedItemCount: 0, failedSourceCount: 0 })).toEqual({
      kind: "info",
      message: BACKLOG_REFRESH_COPY.current,
    });
  });

  it("reports archived tasks", () => {
    expect(backlogRefreshToast({ archivedCount: 1, failedItemCount: 0, failedSourceCount: 0 })).toEqual({
      kind: "success",
      message: BACKLOG_REFRESH_COPY.archivedOne,
    });
    expect(backlogRefreshToast({ archivedCount: 3, failedItemCount: 0, failedSourceCount: 0 })).toEqual({
      kind: "success",
      message: "Archived 3 tasks.",
    });
  });

  it("reports item and source failures separately", () => {
    expect(backlogRefreshToast({ archivedCount: 2, failedItemCount: 1, failedSourceCount: 2 })).toEqual({
      kind: "error",
      message:
        "Archived 2 tasks. SuperPlane could not check 1 intake item. SuperPlane could not connect to 2 intake sources.",
    });
  });
});

import { beforeEach, describe, expect, it } from "vitest";

import {
  LAST_VISITED_CONSOLE_PAGE_STORAGE_KEY,
  readLastVisitedConsolePage,
  recordLastVisitedConsolePage,
  resolveActiveConsolePage,
} from "./lastVisitedConsolePage";

describe("lastVisitedConsolePage", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns null when nothing was recorded", () => {
    expect(readLastVisitedConsolePage("canvas-1")).toBeNull();
  });

  it("records and reads the last visited page per canvas", () => {
    recordLastVisitedConsolePage("canvas-1", "overview");
    recordLastVisitedConsolePage("canvas-2", "details");

    expect(readLastVisitedConsolePage("canvas-1")).toBe("overview");
    expect(readLastVisitedConsolePage("canvas-2")).toBe("details");
  });

  it("overwrites the previous page for the same canvas", () => {
    recordLastVisitedConsolePage("canvas-1", "overview");
    recordLastVisitedConsolePage("canvas-1", "details");

    expect(readLastVisitedConsolePage("canvas-1")).toBe("details");
  });

  it("ignores malformed stored values", () => {
    window.localStorage.setItem(LAST_VISITED_CONSOLE_PAGE_STORAGE_KEY, "not-json");
    expect(readLastVisitedConsolePage("canvas-1")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_CONSOLE_PAGE_STORAGE_KEY, JSON.stringify(["overview"]));
    expect(readLastVisitedConsolePage("canvas-1")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_CONSOLE_PAGE_STORAGE_KEY, JSON.stringify({ "canvas-1": 42 }));
    expect(readLastVisitedConsolePage("canvas-1")).toBeNull();
  });

  it("ignores empty canvas ids and empty page ids", () => {
    recordLastVisitedConsolePage("", "overview");
    recordLastVisitedConsolePage("canvas-1", "");
    expect(readLastVisitedConsolePage("")).toBeNull();
    expect(readLastVisitedConsolePage("canvas-1")).toBeNull();
  });
});

describe("resolveActiveConsolePage", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns null for empty consoles regardless of hints", () => {
    recordLastVisitedConsolePage("canvas-1", "overview");
    expect(
      resolveActiveConsolePage({
        canvasId: "canvas-1",
        pageParam: "overview",
        availablePageIds: [],
      }),
    ).toBeNull();
  });

  it("prefers an explicit ?page= param when it matches", () => {
    recordLastVisitedConsolePage("canvas-1", "details");
    expect(
      resolveActiveConsolePage({
        canvasId: "canvas-1",
        pageParam: "overview",
        availablePageIds: ["overview", "details"],
      }),
    ).toBe("overview");
  });

  it("falls through when ?page= references an unknown page", () => {
    recordLastVisitedConsolePage("canvas-1", "details");
    expect(
      resolveActiveConsolePage({
        canvasId: "canvas-1",
        pageParam: "missing",
        availablePageIds: ["overview", "details"],
      }),
    ).toBe("details");
  });

  it("uses the stored page when no param is set and the id is still valid", () => {
    recordLastVisitedConsolePage("canvas-1", "details");
    expect(
      resolveActiveConsolePage({
        canvasId: "canvas-1",
        pageParam: null,
        availablePageIds: ["overview", "details"],
      }),
    ).toBe("details");
  });

  it("ignores a stale stored page and falls back to the first page", () => {
    recordLastVisitedConsolePage("canvas-1", "removed");
    expect(
      resolveActiveConsolePage({
        canvasId: "canvas-1",
        pageParam: null,
        availablePageIds: ["overview", "details"],
      }),
    ).toBe("overview");
  });

  it("defaults to the first page when nothing else is set", () => {
    expect(
      resolveActiveConsolePage({
        canvasId: "canvas-1",
        pageParam: null,
        availablePageIds: ["overview", "details"],
      }),
    ).toBe("overview");
  });
});

import { describe, expect, it, beforeEach } from "vitest";

import { LAST_VISITED_APP_TAB_STORAGE_KEY, readLastVisitedAppTab, recordLastVisitedAppTab } from "./lastVisitedAppTab";

describe("lastVisitedAppTab", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns null when nothing was recorded", () => {
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();
  });

  it("records and reads the last visited tab per canvas", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    recordLastVisitedAppTab("canvas-2", "files");

    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
    expect(readLastVisitedAppTab("canvas-2")).toBe("files");
  });

  it("overwrites the previous tab for the same canvas", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    recordLastVisitedAppTab("canvas-1", "memory");

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("ignores malformed stored values", () => {
    window.localStorage.setItem(LAST_VISITED_APP_TAB_STORAGE_KEY, "not-json");
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_APP_TAB_STORAGE_KEY, JSON.stringify(["console"]));
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_APP_TAB_STORAGE_KEY, JSON.stringify({ "canvas-1": 42 }));
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();

    window.localStorage.setItem(LAST_VISITED_APP_TAB_STORAGE_KEY, JSON.stringify({ "canvas-1": "invalid" }));
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();
  });

  it("ignores empty canvas ids", () => {
    recordLastVisitedAppTab("", "console");
    expect(readLastVisitedAppTab("")).toBeNull();
  });

  it("ignores tab values outside the allowed set", () => {
    recordLastVisitedAppTab("canvas-1", "invalid" as never);
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();
  });

  it("preserves entries for other canvases when updating one", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    recordLastVisitedAppTab("canvas-2", "files");
    recordLastVisitedAppTab("canvas-1", "memory");

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
    expect(readLastVisitedAppTab("canvas-2")).toBe("files");
  });
});

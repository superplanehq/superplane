import { describe, expect, it, beforeEach } from "vitest";

import {
  loadRecentCanvasOpens,
  recordRecentCanvasOpen,
  RECENT_CANVAS_OPENS_STORAGE_KEY,
  sortCanvasProjectsByRecentOpen,
} from "./recentCanvasOpens";

describe("recentCanvasOpens", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("returns an empty list for unknown organizations", () => {
    expect(loadRecentCanvasOpens("org-1")).toEqual([]);
  });

  it("records opens with the most recent entry first", () => {
    recordRecentCanvasOpen("org-1", "canvas-a");
    const firstPass = recordRecentCanvasOpen("org-1", "canvas-b");

    expect(firstPass.map((entry) => entry.canvasId)).toEqual(["canvas-b", "canvas-a"]);

    const secondPass = recordRecentCanvasOpen("org-1", "canvas-a");
    expect(secondPass.map((entry) => entry.canvasId)).toEqual(["canvas-a", "canvas-b"]);
  });

  it("persists recent opens per organization", () => {
    recordRecentCanvasOpen("org-1", "canvas-a");
    recordRecentCanvasOpen("org-2", "canvas-b");

    expect(loadRecentCanvasOpens("org-1").map((entry) => entry.canvasId)).toEqual(["canvas-a"]);
    expect(loadRecentCanvasOpens("org-2").map((entry) => entry.canvasId)).toEqual(["canvas-b"]);
    expect(window.localStorage.getItem(RECENT_CANVAS_OPENS_STORAGE_KEY)).toContain("org-1");
  });

  it("sorts projects by most recently opened, then name", () => {
    const projects = [
      { id: "alpha", name: "Alpha" },
      { id: "beta", name: "Beta" },
      { id: "gamma", name: "Gamma" },
    ];
    const recentOpens = [
      { canvasId: "gamma", openedAt: 3 },
      { canvasId: "alpha", openedAt: 2 },
    ];

    expect(sortCanvasProjectsByRecentOpen(projects, recentOpens).map((project) => project.id)).toEqual([
      "gamma",
      "alpha",
      "beta",
    ]);
  });
});

import { beforeEach, describe, expect, it } from "vitest";
import {
  CANVAS_APP_PREFERENCES_STORAGE_KEY,
  loadCanvasAppPreferences,
  setCanvasPinned,
  setCanvasStarred,
} from "./canvasAppPreferences";

describe("canvasAppPreferences", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("stores pinned canvases with the most recently pinned first", () => {
    setCanvasPinned("org-1", "user-1", "canvas-a", true);
    const preferences = setCanvasPinned("org-1", "user-1", "canvas-b", true);

    expect(preferences.pinnedCanvasIds).toEqual(["canvas-b", "canvas-a"]);
    expect(loadCanvasAppPreferences("org-1", "user-1").pinnedCanvasIds).toEqual(["canvas-b", "canvas-a"]);
    expect(window.localStorage.getItem(CANVAS_APP_PREFERENCES_STORAGE_KEY)).toContain("org-1:user-1");
  });

  it("stores starred canvases per organization and account", () => {
    setCanvasStarred("org-1", "user-1", "canvas-a", true);
    setCanvasStarred("org-1", "user-2", "canvas-b", true);
    setCanvasStarred("org-2", "user-1", "canvas-c", true);

    expect(loadCanvasAppPreferences("org-1", "user-1").starredCanvasIds).toEqual(["canvas-a"]);
    expect(loadCanvasAppPreferences("org-1", "user-2").starredCanvasIds).toEqual(["canvas-b"]);
    expect(loadCanvasAppPreferences("org-2", "user-1").starredCanvasIds).toEqual(["canvas-c"]);
  });

  it("removes a canvas preference when disabled", () => {
    setCanvasPinned("org-1", "user-1", "canvas-a", true);
    setCanvasStarred("org-1", "user-1", "canvas-a", true);

    expect(setCanvasPinned("org-1", "user-1", "canvas-a", false).pinnedCanvasIds).toEqual([]);
    expect(setCanvasStarred("org-1", "user-1", "canvas-a", false).starredCanvasIds).toEqual([]);
  });
});

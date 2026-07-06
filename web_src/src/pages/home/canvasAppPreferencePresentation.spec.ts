import { describe, expect, it } from "vitest";
import { applyCanvasAppPreferences } from "./canvasAppPreferencePresentation";
import type { CanvasCardData } from "./types";

describe("applyCanvasAppPreferences", () => {
  it("annotates canvases and orders pinned, then starred, then name", () => {
    const canvases = [
      makeCanvas("regular-z", "Zulu"),
      makeCanvas("starred", "Yankee"),
      makeCanvas("newer-starred", "Bravo"),
      makeCanvas("pinned", "Xray"),
      makeCanvas("regular-a", "Alpha"),
    ];

    const result = applyCanvasAppPreferences(canvases, {
      pinnedCanvasIds: ["pinned"],
      starredCanvasIds: ["newer-starred", "starred"],
    });

    expect(result.map((canvas) => canvas.id)).toEqual(["pinned", "newer-starred", "starred", "regular-a", "regular-z"]);
    expect(result.find((canvas) => canvas.id === "pinned")?.isPinned).toBe(true);
    expect(result.find((canvas) => canvas.id === "starred")?.isStarred).toBe(true);
  });
});

function makeCanvas(id: string, name: string): CanvasCardData {
  return {
    id,
    name,
    createdAt: "2026-05-05",
    createdBy: { name: "Ada Lovelace" },
    nodes: [],
    edges: [],
  };
}

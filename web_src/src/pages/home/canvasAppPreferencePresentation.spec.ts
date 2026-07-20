import { describe, expect, it } from "vitest";
import { applyCanvasAppPreferences } from "./canvasAppPreferencePresentation";
import type { CanvasCardData } from "./types";

describe("applyCanvasAppPreferences", () => {
  it("orders starred first by recency, then remaining canvases by name", () => {
    const canvases = [
      makeCanvas("regular-z", "Zulu"),
      makeCanvas("starred", "Yankee", { isStarred: true, starredAt: "2026-05-04T12:00:00Z" }),
      makeCanvas("newer-starred", "Bravo", { isStarred: true, starredAt: "2026-05-05T12:00:00Z" }),
      makeCanvas("regular-a", "Alpha"),
    ];

    const result = applyCanvasAppPreferences(canvases);

    expect(result.map((canvas) => canvas.id)).toEqual(["newer-starred", "starred", "regular-a", "regular-z"]);
    expect(result.find((canvas) => canvas.id === "starred")?.isStarred).toBe(true);
  });
});

function makeCanvas(id: string, name: string, overrides: Partial<CanvasCardData> = {}): CanvasCardData {
  return {
    id,
    name,
    createdAt: "2026-05-05",
    createdBy: { name: "Ada Lovelace" },
    nodes: [],
    edges: [],
    ...overrides,
  };
}

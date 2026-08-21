import { describe, expect, it } from "vitest";

import {
  LINE_BOARD_COLUMN_COLORS,
  lineBoardColumnColorById,
  lineBoardColumnLaneClassName,
} from "./lineBoardColumnColors";

describe("lineBoardColumnColors", () => {
  it("lists ten pastel lane colours for the picker", () => {
    expect(LINE_BOARD_COLUMN_COLORS).toHaveLength(10);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => color.swatchClassName.includes("bg-"))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => color.laneClassName.includes("bg-"))).toBe(true);
  });

  it("resolves a lane class from a colour id", () => {
    expect(lineBoardColumnColorById("lime")?.label).toBe("Lime");
    expect(lineBoardColumnLaneClassName("lime")).toContain("lime");
    expect(lineBoardColumnLaneClassName(null)).toBeUndefined();
  });
});

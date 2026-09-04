import { describe, expect, it } from "vitest";

import {
  LINE_BOARD_COLUMN_COLORS,
  lineBoardColumnColorById,
  lineBoardColumnLaneClassName,
  normalizeColumnColors,
  remapColumnColorsAfterRemovedStep,
  serializeColumnColors,
} from "./lineBoardColumnColors";

describe("lineBoardColumnColors", () => {
  it("lists six colours and uses the same fill for the swatch and the lane", () => {
    expect(LINE_BOARD_COLUMN_COLORS).toHaveLength(6);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => color.className.includes("bg-"))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.map((color) => color.id)).not.toContain("red");
  });

  it("resolves a lane class from a colour id", () => {
    expect(lineBoardColumnColorById("lime")?.label).toBe("Lime");
    expect(lineBoardColumnLaneClassName("lime")).toBe(lineBoardColumnColorById("lime")?.className);
    expect(lineBoardColumnLaneClassName("lime")).toContain("lime");
    expect(lineBoardColumnLaneClassName(null)).toBeUndefined();
  });

  it("normalizes known ids and drops unknown ones", () => {
    expect(normalizeColumnColors({ backlog: "lime", weird: "not-a-color" })).toEqual({ backlog: "lime" });
  });

  it("serializes only columns with an explicit color", () => {
    expect(serializeColumnColors({ backlog: "lime", verify: null, done: "teal" })).toEqual({
      backlog: "lime",
      done: "teal",
    });
  });

  it("shifts later phase colors down after a step is removed", () => {
    expect(
      remapColumnColorsAfterRemovedStep(
        { backlog: "lime", "phase-0": "sky", "phase-1": "teal", "phase-2": "purple", verify: "yellow", done: "slate" },
        1,
      ),
    ).toEqual({
      backlog: "lime",
      "phase-0": "sky",
      "phase-1": "purple",
      verify: "yellow",
      done: "slate",
    });
  });
});

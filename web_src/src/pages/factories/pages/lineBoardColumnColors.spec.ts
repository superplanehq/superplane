import { describe, expect, it } from "bun:test";

import {
  LINE_BOARD_COLUMN_COLORS,
  lineBoardColumnColorById,
  lineBoardColumnLaneClassName,
  lineBoardColumnLaneProps,
  normalizeColumnColors,
  serializeColumnColors,
} from "./lineBoardColumnColors";

describe("lineBoardColumnColors", () => {
  it("lists six colours and uses a dim dark-mode wash on the lane", () => {
    expect(LINE_BOARD_COLUMN_COLORS).toHaveLength(6);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => color.className.includes("bg-"))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => /dark:bg-\S+\/\d+/.test(color.laneClassName))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => /dark:border-\S+\/\d+/.test(color.borderClassName))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.map((color) => color.id)).not.toContain("red");
  });

  it("resolves a lane class from a colour id", () => {
    expect(lineBoardColumnColorById("lime")?.label).toBe("Lime");
    expect(lineBoardColumnLaneClassName("lime")).toBe(lineBoardColumnColorById("lime")?.laneClassName);
    expect(lineBoardColumnLaneClassName("lime")).toContain("lime");
    expect(lineBoardColumnLaneClassName("lime")).toContain("dark:bg-lime-950/40");
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

  it("maps the board view to fill, outline, or no color", () => {
    expect(lineBoardColumnLaneProps("lime", "fill")).toEqual({
      surfaceClassName: lineBoardColumnColorById("lime")?.laneClassName,
    });
    expect(lineBoardColumnLaneProps("lime", "off", { mutedFallback: true })).toEqual({ className: "bg-muted" });
    expect(lineBoardColumnLaneProps("lime", "borders", { mutedFallback: true })).toEqual({
      className: `bg-muted ${lineBoardColumnColorById("lime")?.borderClassName}`,
    });
    expect(lineBoardColumnLaneProps(null, "fill", { mutedFallback: true })).toEqual({ className: "bg-muted" });
  });
});

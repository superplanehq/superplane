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
  it("lists ten colours and uses a quieter wash than vivid in both themes", () => {
    expect(LINE_BOARD_COLUMN_COLORS).toHaveLength(10);
    expect(LINE_BOARD_COLUMN_COLORS.map((color) => color.id)).toEqual([
      "emerald",
      "lime",
      "yellow",
      "orange",
      "rose",
      "teal",
      "sky",
      "blue",
      "purple",
      "slate",
    ]);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => color.className.includes("bg-"))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => color.laneClassName !== color.className)).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => /bg-\S+-100/.test(color.laneClassName))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => /dark:bg-\S+\/\d+/.test(color.laneClassName))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.every((color) => /dark:border-\S+\/\d+/.test(color.borderClassName))).toBe(true);
    expect(LINE_BOARD_COLUMN_COLORS.map((color) => color.id)).not.toContain("red");
  });

  it("resolves orange and rose to a wash, a vivid fill, and a border", () => {
    const orange = lineBoardColumnColorById("orange");
    const rose = lineBoardColumnColorById("rose");

    expect(orange?.label).toBe("Orange");
    expect(orange?.laneClassName).toBe("bg-orange-100 dark:bg-orange-950/40");
    expect(orange?.className).toBe("bg-orange-300 dark:bg-orange-800");
    expect(orange?.borderClassName).toBe("border-orange-400 dark:border-orange-800/45");

    expect(rose?.label).toBe("Rose");
    expect(rose?.laneClassName).toBe("bg-rose-100 dark:bg-rose-950/40");
    expect(rose?.className).toBe("bg-rose-300 dark:bg-rose-800");
    expect(rose?.borderClassName).toBe("border-rose-400 dark:border-rose-800/45");
  });

  it("resolves emerald and blue to a wash, a vivid fill, and a border", () => {
    const emerald = lineBoardColumnColorById("emerald");
    const blue = lineBoardColumnColorById("blue");

    expect(emerald?.label).toBe("Emerald");
    expect(emerald?.laneClassName).toBe("bg-emerald-100 dark:bg-emerald-950/40");
    expect(emerald?.className).toBe("bg-emerald-300 dark:bg-emerald-800");
    expect(emerald?.borderClassName).toBe("border-emerald-400 dark:border-emerald-800/45");

    expect(blue?.label).toBe("Blue");
    expect(blue?.laneClassName).toBe("bg-blue-100 dark:bg-blue-950/40");
    expect(blue?.className).toBe("bg-blue-300 dark:bg-blue-800");
    expect(blue?.borderClassName).toBe("border-blue-400 dark:border-blue-800/45");
  });

  it("resolves a lane class from a colour id", () => {
    expect(lineBoardColumnColorById("lime")?.label).toBe("Lime");
    expect(lineBoardColumnLaneClassName("lime")).toBe(lineBoardColumnColorById("lime")?.laneClassName);
    expect(lineBoardColumnLaneClassName("lime")).toContain("bg-lime-100");
    expect(lineBoardColumnLaneClassName("lime")).toContain("dark:bg-lime-950/40");
    expect(lineBoardColumnLaneClassName("rose")).toContain("bg-rose-100");
    expect(lineBoardColumnLaneClassName("orange")).toContain("bg-orange-100");
    expect(lineBoardColumnLaneClassName("emerald")).toContain("bg-emerald-100");
    expect(lineBoardColumnLaneClassName("blue")).toContain("bg-blue-100");
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

  it("maps the board view to vivid, dim, outline, or no color", () => {
    expect(lineBoardColumnLaneProps("lime", "dim")).toEqual({
      surfaceClassName: lineBoardColumnColorById("lime")?.laneClassName,
    });
    expect(lineBoardColumnLaneProps("lime", "vivid")).toEqual({
      surfaceClassName: lineBoardColumnColorById("lime")?.className,
    });
    expect(lineBoardColumnLaneProps("lime", "off", { mutedFallback: true })).toEqual({ className: "bg-muted" });
    expect(lineBoardColumnLaneProps("lime", "borders", { mutedFallback: true })).toEqual({
      className: `bg-muted ${lineBoardColumnColorById("lime")?.borderClassName}`,
    });
    expect(lineBoardColumnLaneProps(null, "dim", { mutedFallback: true })).toEqual({ className: "bg-muted" });
  });
});

import { describe, expect, it } from "vitest";

import { makeRowStyleResolver, ROW_STYLE_CLASS } from "./rowStyles";
import type { WidgetRowStyle } from "./types";

describe("makeRowStyleResolver", () => {
  it("returns undefined when no rules are configured", () => {
    expect(makeRowStyleResolver(undefined)).toBeUndefined();
    expect(makeRowStyleResolver([])).toBeUndefined();
  });

  it("returns the first matching rule's tone class (first-match-wins)", () => {
    const rules: WidgetRowStyle[] = [
      { field: "status", op: "eq", value: "error", tone: "red-soft" },
      { field: "status", op: "eq", value: "deploying", tone: "orange-soft" },
    ];
    const resolve = makeRowStyleResolver(rules);
    expect(resolve).toBeDefined();
    expect(resolve!({ status: "error" })).toBe(ROW_STYLE_CLASS["red-soft"]);
    expect(resolve!({ status: "deploying" })).toBe(ROW_STYLE_CLASS["orange-soft"]);
  });

  it("returns undefined when no rule matches the row", () => {
    const rules: WidgetRowStyle[] = [{ field: "status", op: "eq", value: "error", tone: "red" }];
    const resolve = makeRowStyleResolver(rules)!;
    expect(resolve({ status: "passed" })).toBeUndefined();
  });

  it("earlier rules take precedence when multiple match the same row", () => {
    const rules: WidgetRowStyle[] = [
      { field: "status", op: "contains", value: "err", tone: "red" },
      { field: "status", op: "eq", value: "error", tone: "green" },
    ];
    const resolve = makeRowStyleResolver(rules)!;
    // Both rules match `status === "error"`, but the first wins.
    expect(resolve({ status: "error" })).toBe(ROW_STYLE_CLASS.red);
  });

  it("supports exists/not_exists operators with no value", () => {
    const exists: WidgetRowStyle = { field: "deployedAt", op: "exists", tone: "blue-soft" };
    const notExists: WidgetRowStyle = { field: "deployedAt", op: "not_exists", tone: "dimmed" };

    const resolveExists = makeRowStyleResolver([exists])!;
    expect(resolveExists({ deployedAt: "2026-01-01" })).toBe(ROW_STYLE_CLASS["blue-soft"]);
    expect(resolveExists({})).toBeUndefined();

    const resolveMissing = makeRowStyleResolver([notExists])!;
    expect(resolveMissing({})).toBe(ROW_STYLE_CLASS.dimmed);
    expect(resolveMissing({ deployedAt: "x" })).toBeUndefined();
  });

  it("maps soft tones to lighter backgrounds than full tones", () => {
    expect(ROW_STYLE_CLASS["red-soft"]).toBe("bg-red-50");
    expect(ROW_STYLE_CLASS.red).toBe("bg-red-100");
    expect(ROW_STYLE_CLASS["red-soft"]).not.toBe(ROW_STYLE_CLASS.red);
  });
});

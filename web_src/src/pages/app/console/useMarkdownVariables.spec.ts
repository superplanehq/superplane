import { describe, expect, it } from "vitest";

import { pickMemoryRows } from "./useMarkdownVariables";
import type { MarkdownMemoryVariableSource } from "./markdownVariables";

function memorySource(extra: Partial<MarkdownMemoryVariableSource>): MarkdownMemoryVariableSource {
  return { kind: "memory", namespace: "ns", ...extra };
}

const rows = [{ name: "a" }, { name: "b" }, { name: "c" }];

describe("pickMemoryRows", () => {
  it("returns the first row in single mode (default)", () => {
    expect(pickMemoryRows(rows, memorySource({}))).toEqual({ name: "a" });
    expect(pickMemoryRows(rows, memorySource({ mode: "single" }))).toEqual({ name: "a" });
  });

  it("returns the full sorted array in list mode with no limit", () => {
    expect(pickMemoryRows(rows, memorySource({ mode: "list" }))).toEqual(rows);
  });

  it("respects an explicit limit in list mode", () => {
    expect(pickMemoryRows(rows, memorySource({ mode: "list", limit: 2 }))).toEqual([{ name: "a" }, { name: "b" }]);
  });

  it("ignores limit when not in list mode", () => {
    // Single mode authors should keep getting the first row even if a stale
    // `limit` is still present on the source.
    expect(pickMemoryRows(rows, memorySource({ limit: 99 }))).toEqual({ name: "a" });
  });

  it("returns the full sorted array when limit is zero, negative, or fractional", () => {
    // Validation is layered above this helper - here we mirror the
    // production resolver's fail-soft behavior: any non-positive-integer
    // limit means "no cap", so the panel still renders the full list
    // instead of an empty one.
    expect(pickMemoryRows(rows, memorySource({ mode: "list", limit: 0 }))).toEqual(rows);
    expect(pickMemoryRows(rows, memorySource({ mode: "list", limit: -1 }))).toEqual(rows);
  });

  it("returns an empty array when the sorted set is empty in list mode", () => {
    expect(pickMemoryRows([], memorySource({ mode: "list" }))).toEqual([]);
  });

  it("returns undefined when the sorted set is empty in single mode", () => {
    // Mirrors how `sorted[0]` behaves; callers should still gate on the
    // outer `resolveMemoryVariable` empty-array branch instead of using
    // this helper for the no-rows path.
    expect(pickMemoryRows([], memorySource({}))).toBeUndefined();
  });
});

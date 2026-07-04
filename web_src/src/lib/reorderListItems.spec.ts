import { describe, expect, it } from "vitest";

import { reorderListItems } from "./reorderListItems";

describe("reorderListItems", () => {
  it("moves an item to a new index", () => {
    expect(reorderListItems(["a", "b", "c"], 0, 2)).toEqual(["b", "c", "a"]);
  });

  it("returns the same array when indices are equal or out of range", () => {
    const items = ["a", "b"];
    expect(reorderListItems(items, 0, 0)).toBe(items);
    expect(reorderListItems(items, -1, 1)).toBe(items);
    expect(reorderListItems(items, 0, 5)).toBe(items);
  });
});

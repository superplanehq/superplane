import { describe, expect, it } from "bun:test";

import { stabilizeItemOrder } from "./stabilizeItemOrder";

function idOf(item: { id: string }): string {
  return item.id;
}

describe("stabilizeItemOrder", () => {
  it("returns the incoming list when there is no previous order", () => {
    const items = [{ id: "b" }, { id: "a" }];
    expect(stabilizeItemOrder([], items, idOf)).toEqual(items);
  });

  it("keeps the previous place when the same ids come back in a new sort", () => {
    const previousIds = ["newer", "older"];
    const resorted = [
      { id: "older", updatedAt: "16:00" },
      { id: "newer", updatedAt: "12:00" },
    ];

    expect(stabilizeItemOrder(previousIds, resorted, idOf).map((item) => item.id)).toEqual(["newer", "older"]);
  });

  it("drops ids that left and puts a single new id first", () => {
    const previousIds = ["a", "b", "c"];
    const next = [{ id: "d" }, { id: "c" }, { id: "a" }];

    expect(stabilizeItemOrder(previousIds, next, idOf).map((item) => item.id)).toEqual(["d", "a", "c"]);
  });

  it("keeps the incoming sort when many new ids appear", () => {
    const previousIds = ["a"];
    const next = [{ id: "c" }, { id: "b" }, { id: "a" }];

    expect(stabilizeItemOrder(previousIds, next, idOf).map((item) => item.id)).toEqual(["c", "b", "a"]);
  });
});

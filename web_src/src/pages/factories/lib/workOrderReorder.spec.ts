import { describe, expect, it } from "vitest";
import { applyWorkOrderReorderMove, buildWorkOrderReorderMove } from "./workOrderReorder";

describe("buildWorkOrderReorderMove", () => {
  const laneIds = ["a", "b", "c", "d"];

  it("returns null when dropped on itself", () => {
    expect(buildWorkOrderReorderMove(laneIds, "b", "b")).toBeNull();
  });

  it("returns null when the active id isn't in the lane", () => {
    expect(buildWorkOrderReorderMove(laneIds, "missing", "b")).toBeNull();
  });

  it("returns null when the over id isn't in the lane", () => {
    expect(buildWorkOrderReorderMove(laneIds, "b", "missing")).toBeNull();
  });

  it("moves a card down: neighbors become the card it passed and the one after", () => {
    // a, b, c, d -> b, c, a, d
    expect(buildWorkOrderReorderMove(laneIds, "a", "c")).toEqual({
      orderId: "a",
      previousOrderId: "c",
      nextOrderId: "d",
    });
  });

  it("moves a card up: neighbors become the one before and the card it passed", () => {
    // a, b, c, d -> a, d, b, c
    expect(buildWorkOrderReorderMove(laneIds, "d", "b")).toEqual({
      orderId: "d",
      previousOrderId: "a",
      nextOrderId: "b",
    });
  });

  it("moving to the very top drops the previous neighbor", () => {
    expect(buildWorkOrderReorderMove(laneIds, "d", "a")).toEqual({
      orderId: "d",
      previousOrderId: undefined,
      nextOrderId: "a",
    });
  });

  it("moving to the very bottom drops the next neighbor", () => {
    expect(buildWorkOrderReorderMove(laneIds, "a", "d")).toEqual({
      orderId: "a",
      previousOrderId: "d",
      nextOrderId: undefined,
    });
  });
});

describe("applyWorkOrderReorderMove", () => {
  const items = [{ id: "a" }, { id: "b" }, { id: "c" }, { id: "d" }];

  it("moves the item next to its new previous neighbor", () => {
    const next = applyWorkOrderReorderMove(items, { orderId: "a", previousOrderId: "c" });
    expect(next.map((i) => i.id)).toEqual(["b", "c", "a", "d"]);
  });

  it("moves the item next to its new next neighbor", () => {
    const next = applyWorkOrderReorderMove(items, { orderId: "d", nextOrderId: "b" });
    expect(next.map((i) => i.id)).toEqual(["a", "d", "b", "c"]);
  });

  it("moves the item to the top when there is no previous neighbor", () => {
    const next = applyWorkOrderReorderMove(items, { orderId: "c", nextOrderId: "a" });
    expect(next.map((i) => i.id)).toEqual(["c", "a", "b", "d"]);
  });

  it("moves the item to the bottom when there is no next neighbor", () => {
    const next = applyWorkOrderReorderMove(items, { orderId: "a", previousOrderId: "d" });
    expect(next.map((i) => i.id)).toEqual(["b", "c", "d", "a"]);
  });

  it("is a no-op when the moved id isn't found", () => {
    const next = applyWorkOrderReorderMove(items, { orderId: "missing", previousOrderId: "a" });
    expect(next).toBe(items);
  });
});

import { describe, expect, it } from "vitest";
import { pickInitialFactory } from "./lastVisitedFactory";

const FACTORIES = [
  { id: "factory-1", key: "AA" },
  { id: "factory-2", key: "BB" },
];

describe("pickInitialFactory", () => {
  it("prefers the last-visited factory when it still exists", () => {
    expect(pickInitialFactory(FACTORIES, "factory-2")).toBe(FACTORIES[1]);
  });

  it("falls back to the first factory when there is no last-visited match", () => {
    expect(pickInitialFactory(FACTORIES, "missing")).toBe(FACTORIES[0]);
    expect(pickInitialFactory(FACTORIES, null)).toBe(FACTORIES[0]);
  });

  it("returns null when there are no factories", () => {
    expect(pickInitialFactory([], "factory-1")).toBeNull();
  });
});

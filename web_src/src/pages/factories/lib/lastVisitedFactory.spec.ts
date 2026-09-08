import { describe, expect, it } from "vitest";
import { pickInitialFactory, pickReadyFactory } from "./lastVisitedFactory";

const FACTORIES = [
  { id: "factory-1", key: "AA" },
  { id: "factory-2", key: "BB" },
];

const MIXED_FACTORIES = [
  { id: "factory-setup", key: "SETUP", onboarding: {} },
  { id: "factory-ready", key: "READY", onboarding: { completedAt: "2026-09-01T00:00:00.000Z" } },
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

describe("pickReadyFactory", () => {
  it("skips an incomplete last-visited workspace when a ready one exists", () => {
    expect(pickReadyFactory(MIXED_FACTORIES, "factory-setup")).toBe(MIXED_FACTORIES[1]);
  });

  it("keeps an incomplete workspace when it is the only one", () => {
    expect(pickReadyFactory([MIXED_FACTORIES[0]], "factory-setup")).toBe(MIXED_FACTORIES[0]);
  });
});

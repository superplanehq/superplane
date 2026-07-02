import { describe, expect, it } from "vitest";
import { computeFitViewContentKey, shouldRefitOnInit, stampFittedContentKey } from "./fitView";

describe("computeFitViewContentKey", () => {
  it("combines canvas id and view key outside run inspection", () => {
    expect(computeFitViewContentKey({ isRunInspectionMode: false, canvasId: "c1", canvasViewKey: "v1" })).toBe("c1:v1");
  });

  it("returns undefined in run inspection so run mode keeps its own fit handling", () => {
    expect(
      computeFitViewContentKey({ isRunInspectionMode: true, canvasId: "c1", canvasViewKey: "v1" }),
    ).toBeUndefined();
  });

  it("tolerates a missing canvas id", () => {
    expect(computeFitViewContentKey({ isRunInspectionMode: false, canvasViewKey: "live" })).toBe(":live");
  });
});

describe("shouldRefitOnInit", () => {
  it("fits on the first initialization", () => {
    expect(shouldRefitOnInit({ hasFittedBefore: false, fitViewContentKey: "c1:v1", lastFittedContentKey: null })).toBe(
      true,
    );
  });

  it("re-fits when the displayed content key changed", () => {
    expect(
      shouldRefitOnInit({ hasFittedBefore: true, fitViewContentKey: "c1:v2", lastFittedContentKey: "c1:v1" }),
    ).toBe(true);
  });

  it("restores instead of re-fitting when the content key is unchanged", () => {
    expect(
      shouldRefitOnInit({ hasFittedBefore: true, fitViewContentKey: "c1:v1", lastFittedContentKey: "c1:v1" }),
    ).toBe(false);
  });

  it("does not force a re-fit when there is no content key (run inspection)", () => {
    expect(
      shouldRefitOnInit({ hasFittedBefore: true, fitViewContentKey: undefined, lastFittedContentKey: "c1:v1" }),
    ).toBe(false);
  });
});

describe("stampFittedContentKey", () => {
  it("records the fitted content key", () => {
    const ref = { current: null as string | null };
    stampFittedContentKey(ref, "c1:v1");
    expect(ref.current).toBe("c1:v1");
  });

  it("ignores an undefined content key so a later init re-fits", () => {
    const ref = { current: "c1:v1" as string | null };
    stampFittedContentKey(ref, undefined);
    expect(ref.current).toBe("c1:v1");
  });

  it("is a no-op without a ref", () => {
    expect(() => stampFittedContentKey(undefined, "c1:v1")).not.toThrow();
  });
});

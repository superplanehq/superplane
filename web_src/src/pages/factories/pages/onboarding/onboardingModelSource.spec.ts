import { beforeEach, describe, expect, it } from "bun:test";

import { readOnboardingModelSource, writeOnboardingModelSource } from "./onboardingModelSource";

describe("onboardingModelSource", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("stores the model source for a factory", () => {
    writeOnboardingModelSource("factory-1", "hosted");

    expect(readOnboardingModelSource("factory-1")).toBe("hosted");
    expect(readOnboardingModelSource("factory-2")).toBeNull();
  });

  it("ignores values that are not a model source", () => {
    localStorage.setItem("superplane:onboarding-model-source:factory-1", "other");

    expect(readOnboardingModelSource("factory-1")).toBeNull();
  });

  it("does nothing without a factory", () => {
    writeOnboardingModelSource("", "own-key");

    expect(readOnboardingModelSource("")).toBeNull();
  });
});

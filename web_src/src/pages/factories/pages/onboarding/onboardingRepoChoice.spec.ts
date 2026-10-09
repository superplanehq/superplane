import { beforeEach, describe, expect, it } from "bun:test";

import { clearOnboardingRepoChoice, readOnboardingRepoChoice, writeOnboardingRepoChoice } from "./onboardingRepoChoice";

describe("onboardingRepoChoice", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("stores the selected repository for one factory", () => {
    writeOnboardingRepoChoice("factory-1", "acme/payments");

    expect(readOnboardingRepoChoice("factory-1")).toBe("acme/payments");
    expect(readOnboardingRepoChoice("factory-2")).toBeNull();
  });

  it("ignores a blank factory or repository", () => {
    writeOnboardingRepoChoice("", "acme/payments");
    writeOnboardingRepoChoice("factory-1", "   ");

    expect(readOnboardingRepoChoice("")).toBeNull();
    expect(readOnboardingRepoChoice("factory-1")).toBeNull();
  });

  it("removes the stored repository", () => {
    writeOnboardingRepoChoice("factory-1", "acme/payments");
    clearOnboardingRepoChoice("factory-1");

    expect(readOnboardingRepoChoice("factory-1")).toBeNull();
  });
});

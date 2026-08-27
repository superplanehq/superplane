import { describe, expect, it } from "vitest";

import { canStartFactoryConfigureDiscard, resolveDraftTitleToPersist } from "./useFactoryAppConfigureTitle";

describe("resolveDraftTitleToPersist", () => {
  it("returns null when the draft matches the saved title", () => {
    expect(resolveDraftTitleToPersist("Same", "Same")).toBeNull();
    expect(resolveDraftTitleToPersist(null, "Same")).toBeNull();
  });

  it("returns the trimmed draft when it differs", () => {
    expect(resolveDraftTitleToPersist("  New name  ", "Old")).toBe("New name");
  });
});

describe("canStartFactoryConfigureDiscard", () => {
  it("is false when discard is missing so loading chrome cannot leave without reset", () => {
    expect(
      canStartFactoryConfigureDiscard({
        configureBusy: false,
        renamePending: false,
        hasDiscardAction: false,
      }),
    ).toBe(false);
  });

  it("is false while save or rename is pending", () => {
    expect(
      canStartFactoryConfigureDiscard({
        configureBusy: true,
        renamePending: false,
        hasDiscardAction: true,
      }),
    ).toBe(false);
    expect(
      canStartFactoryConfigureDiscard({
        configureBusy: false,
        renamePending: true,
        hasDiscardAction: true,
      }),
    ).toBe(false);
  });

  it("is true when AppPage discard is ready", () => {
    expect(
      canStartFactoryConfigureDiscard({
        configureBusy: false,
        renamePending: false,
        hasDiscardAction: true,
      }),
    ).toBe(true);
  });
});

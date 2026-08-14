import { describe, expect, it } from "vitest";

import { resolveDraftTitleToPersist } from "./useFactoryAppConfigureTitle";

describe("resolveDraftTitleToPersist", () => {
  it("returns null when the draft matches the saved title", () => {
    expect(resolveDraftTitleToPersist("Same", "Same")).toBeNull();
    expect(resolveDraftTitleToPersist(null, "Same")).toBeNull();
  });

  it("returns the trimmed draft when it differs", () => {
    expect(resolveDraftTitleToPersist("  New name  ", "Old")).toBe("New name");
  });
});

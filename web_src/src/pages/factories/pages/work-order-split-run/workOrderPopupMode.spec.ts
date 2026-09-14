import { describe, expect, it } from "vitest";

import { workOrderPopupMode } from "./workOrderPopupMode";

describe("workOrderPopupMode", () => {
  it("uses the classic popup when refinement is disabled", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: false,
        hasLookupIdentity: true,
      }),
    ).toBe("classic");
  });

  it("uses the analysis popup for a pinned session after the flag is disabled", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: true,
        refinementEnabled: false,
        hasLookupIdentity: true,
      }),
    ).toBe("analysis");
  });

  it("uses the analysis popup for a historical task without analysis", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: true,
        hasLookupIdentity: true,
      }),
    ).toBe("analysis");
  });

  it("keeps incomplete fixture identities in analysis demo mode", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: false,
        hasLookupIdentity: false,
      }),
    ).toBe("analysis");
  });
});

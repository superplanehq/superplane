import { describe, expect, it } from "vitest";

import { workOrderPopupMode } from "./workOrderPopupMode";

describe("workOrderPopupMode", () => {
  it("uses the classic popup when refinement is disabled", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: false,
        analysisActive: true,
        hasLookupIdentity: true,
      }),
    ).toBe("classic");
  });

  it("uses the analysis popup for a pinned session after the flag is disabled", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: true,
        refinementEnabled: false,
        analysisActive: false,
        hasLookupIdentity: true,
      }),
    ).toBe("analysis");
  });

  it("uses the analysis popup while enabled analysis is active", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: true,
        analysisActive: true,
        hasLookupIdentity: true,
      }),
    ).toBe("analysis");
  });

  it("keeps a draft that never had a refinement session in classic mode", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: true,
        analysisActive: false,
        hasLookupIdentity: true,
      }),
    ).toBe("classic");
  });

  it("keeps incomplete fixture identities in analysis demo mode", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: false,
        analysisActive: false,
        hasLookupIdentity: false,
      }),
    ).toBe("analysis");
  });
});

import { describe, expect, it } from "vitest";

import { workOrderPopupMode } from "./workOrderPopupMode";

describe("workOrderPopupMode", () => {
  it("waits for Task Refinement access before choosing a popup", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: false,
        refinementLoading: true,
        analysisActive: false,
        hasLookupIdentity: true,
      }),
    ).toBe("loading");
  });

  it("waits for the session lookup when an existing session can change the popup", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        refinementEnabled: true,
        sessionLoading: true,
        analysisActive: false,
        hasLookupIdentity: true,
      }),
    ).toBe("loading");
  });

  it("shows known analysis results while the session lookup continues", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        hasAnalysisResult: true,
        refinementEnabled: true,
        sessionLoading: true,
        analysisActive: false,
        hasLookupIdentity: true,
      }),
    ).toBe("analysis");
  });

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
        hasAnalysisResult: false,
        refinementEnabled: true,
        analysisActive: false,
        hasLookupIdentity: true,
        isDraft: true,
      }),
    ).toBe("classic");
  });

  it("uses the analysis popup for a started task while artifacts load", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        hasAnalysisResult: false,
        refinementEnabled: true,
        artifactsLoading: true,
        analysisActive: false,
        hasLookupIdentity: true,
        isDraft: false,
      }),
    ).toBe("analysis");
  });

  it("waits for artifacts before keeping an unrefined draft in classic mode", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        hasAnalysisResult: false,
        refinementEnabled: true,
        artifactsLoading: true,
        analysisActive: false,
        hasLookupIdentity: true,
        isDraft: true,
      }),
    ).toBe("loading");
  });

  it("uses the classic popup when refinement is disabled even if a score exists", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        hasAnalysisResult: true,
        refinementEnabled: false,
        analysisActive: false,
        hasLookupIdentity: true,
      }),
    ).toBe("classic");
  });

  it("keeps the analysis popup after Start when a score or spec exists", () => {
    expect(
      workOrderPopupMode({
        hasPlanningSession: false,
        hasAnalysisResult: true,
        refinementEnabled: true,
        analysisActive: false,
        hasLookupIdentity: true,
      }),
    ).toBe("analysis");
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

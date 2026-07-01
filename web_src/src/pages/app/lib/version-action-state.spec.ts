import { describe, expect, it } from "vitest";
import {
  getVersionActionAvailability,
  hasMergeableBranchChanges,
  hasVersionActionChanges,
} from "./version-action-state";

describe("getVersionActionAvailability", () => {
  it("keeps publish enabled while local draft changes are still being saved", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("keeps publish enabled after local save activity has settled but draft changes are still pending", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("disables publish while publish is pending", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: true,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(true);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("disables publish when the latest draft matches live", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: false,
    });

    expect(result.publishVersionDisabled).toBe(true);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("enables merge when a feature branch head differs from live without canvas/console diff", () => {
    const result = getVersionActionAvailability({
      hasEditableVersion: true,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: false,
      hasMergeableBranchChanges: true,
    });

    expect(result.publishVersionDisabled).toBe(false);
  });
});

describe("hasMergeableBranchChanges", () => {
  it("returns false on main", () => {
    expect(
      hasMergeableBranchChanges({
        isMainBranch: true,
        branchHeadVersionId: "branch-head",
        liveVersionId: "live",
      }),
    ).toBe(false);
  });

  it("returns true when feature branch head differs from live", () => {
    expect(
      hasMergeableBranchChanges({
        isMainBranch: false,
        branchHeadVersionId: "branch-head",
        liveVersionId: "live",
      }),
    ).toBe(true);
  });

  it("returns false when branch head matches live", () => {
    expect(
      hasMergeableBranchChanges({
        isMainBranch: false,
        branchHeadVersionId: "same",
        liveVersionId: "same",
      }),
    ).toBe(false);
  });
});

describe("hasVersionActionChanges", () => {
  it("includes mergeable branch changes", () => {
    expect(
      hasVersionActionChanges({
        hasDraftDiffVersusLive: false,
        hasMergeableBranchChanges: true,
      }),
    ).toBe(true);
  });
});

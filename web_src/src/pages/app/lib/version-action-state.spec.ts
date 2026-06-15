import { describe, expect, it } from "vitest";
import { getDraftChangeIndicators, getVersionActionAvailability } from "./version-action-state";

describe("getVersionActionAvailability", () => {
  it("keeps publish enabled while local draft changes are still being saved", () => {
    const result = getVersionActionAvailability({
      isChangeManagementDisabled: true,
      hasEditableVersion: true,
      createChangeRequestPending: false,
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
      isChangeManagementDisabled: true,
      hasEditableVersion: true,
      createChangeRequestPending: false,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it("keeps change request creation enabled when draft changes are pending locally", () => {
    const result = getVersionActionAvailability({
      isChangeManagementDisabled: false,
      hasEditableVersion: true,
      createChangeRequestPending: false,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.createChangeRequestDisabled).toBe(false);
    expect(result.createChangeRequestDisabledTooltip).toBeUndefined();
    expect(result.publishVersionDisabled).toBe(false);
    expect(result.publishVersionDisabledTooltip).toBeUndefined();
  });

  it.each([true, false])(
    "disables publish/propose when the latest draft matches live (change management %s)",
    (isChangeManagementDisabled) => {
      const result = getVersionActionAvailability({
        isChangeManagementDisabled,
        hasEditableVersion: true,
        createChangeRequestPending: false,
        publishPending: false,
        canvasDeletedRemotely: false,
        isPreparingVersionAction: false,
        hasDraftDiffVersusLive: false,
      });

      expect(result.publishVersionDisabled).toBe(true);
      expect(result.publishVersionDisabledTooltip).toBeUndefined();
    },
  );

  // Regression: committing a repository file (e.g. README.md) to the draft is a
  // publishable change even when the canvas graph and console specs are unchanged.
  // The Publish button must stay enabled.
  it("keeps publish enabled when only repository files differ from live", () => {
    const result = getVersionActionAvailability({
      isChangeManagementDisabled: true,
      hasEditableVersion: true,
      createChangeRequestPending: false,
      publishPending: false,
      canvasDeletedRemotely: false,
      isPreparingVersionAction: false,
      hasDraftDiffVersusLive: true,
    });

    expect(result.publishVersionDisabled).toBe(false);
  });
});

describe("getDraftChangeIndicators", () => {
  // Regression: a draft whose only difference from live is a committed repository
  // file must light the global unpublished-changes indicator and the Files tab's
  // committed (blue) dot, even though the graph/console specs match live.
  it("flags unpublished file changes when only repository files differ from live", () => {
    const result = getDraftChangeIndicators({
      suppressUnpublishedDraftDiscard: false,
      hasLatestDraftVersion: true,
      hasDraftGraphDiffVersusLive: false,
      hasDraftConsoleDiffVersusLive: false,
      hasDraftFilesDiffVersusLive: true,
      hasDraftDiffVersusLive: true,
    });

    expect(result.hasUnpublishedDraftChanges).toBe(true);
    expect(result.hasUnpublishedFilesDraftChanges).toBe(true);
    expect(result.hasUnpublishedCanvasDraftChanges).toBe(false);
    expect(result.hasUnpublishedConsoleDraftChanges).toBe(false);
  });

  it("reports no unpublished changes when the draft matches live across all surfaces", () => {
    const result = getDraftChangeIndicators({
      suppressUnpublishedDraftDiscard: false,
      hasLatestDraftVersion: true,
      hasDraftGraphDiffVersusLive: false,
      hasDraftConsoleDiffVersusLive: false,
      hasDraftFilesDiffVersusLive: false,
      hasDraftDiffVersusLive: false,
    });

    expect(result.hasUnpublishedDraftChanges).toBe(false);
    expect(result.hasUnpublishedFilesDraftChanges).toBe(false);
  });
});

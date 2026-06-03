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
});

describe("getDraftChangeIndicators", () => {
  it("returns no indicators when draft discard is suppressed", () => {
    expect(
      getDraftChangeIndicators({
        suppressUnpublishedDraftDiscard: true,
        hasLatestDraftVersion: true,
        hasDraftGraphDiffVersusLive: true,
        hasDraftConsoleDiffVersusLive: false,
        hasDraftDiffVersusLive: true,
        hasCanvasStagingChanges: true,
        hasConsoleStagingChanges: false,
      }),
    ).toMatchObject({
      hasUncommittedCanvasDraftChanges: false,
      hasCommittedCanvasDraftChanges: false,
      hasUnpublishedCanvasDraftChanges: false,
    });
  });

  it("splits uncommitted staging from committed draft-vs-live per surface", () => {
    expect(
      getDraftChangeIndicators({
        suppressUnpublishedDraftDiscard: false,
        hasLatestDraftVersion: true,
        hasDraftGraphDiffVersusLive: true,
        hasDraftConsoleDiffVersusLive: false,
        hasDraftDiffVersusLive: true,
        hasCanvasStagingChanges: true,
        hasConsoleStagingChanges: false,
      }),
    ).toEqual({
      hasUnpublishedDraftChanges: true,
      hasUnpublishedCanvasDraftChanges: true,
      hasUnpublishedConsoleDraftChanges: false,
      hasUnpublishedFilesDraftChanges: false,
      hasUncommittedCanvasDraftChanges: true,
      hasUncommittedConsoleDraftChanges: false,
      hasUncommittedFilesDraftChanges: false,
      hasCommittedCanvasDraftChanges: true,
      hasCommittedConsoleDraftChanges: false,
      hasUncommittedDraftChanges: true,
      hasCommittedDraftChanges: true,
      readyToPublishDraftChanges: false,
      readyToPublishCanvasDraftChanges: false,
      readyToPublishConsoleDraftChanges: false,
    });
  });

  it("includes repository file staging in uncommitted draft indicators", () => {
    expect(
      getDraftChangeIndicators({
        suppressUnpublishedDraftDiscard: false,
        hasLatestDraftVersion: true,
        hasDraftGraphDiffVersusLive: false,
        hasDraftConsoleDiffVersusLive: false,
        hasDraftDiffVersusLive: false,
        hasCanvasStagingChanges: false,
        hasConsoleStagingChanges: false,
        hasFilesStagingChanges: true,
      }),
    ).toMatchObject({
      hasUncommittedFilesDraftChanges: true,
      hasUnpublishedFilesDraftChanges: true,
      hasUncommittedDraftChanges: true,
      hasUnpublishedDraftChanges: true,
    });
  });

  it("hides ready-to-publish indicators while uncommitted changes exist", () => {
    const indicators = getDraftChangeIndicators({
      suppressUnpublishedDraftDiscard: false,
      hasLatestDraftVersion: true,
      hasDraftGraphDiffVersusLive: true,
      hasDraftConsoleDiffVersusLive: false,
      hasDraftDiffVersusLive: true,
      hasCanvasStagingChanges: true,
      hasConsoleStagingChanges: false,
    });

    expect(indicators.hasCommittedDraftChanges).toBe(true);
    expect(indicators.hasUncommittedDraftChanges).toBe(true);
    expect(indicators.readyToPublishDraftChanges).toBe(false);
    expect(indicators.readyToPublishCanvasDraftChanges).toBe(false);
  });

  it("shows only orange when staging differs but branch matches live", () => {
    expect(
      getDraftChangeIndicators({
        suppressUnpublishedDraftDiscard: false,
        hasLatestDraftVersion: true,
        hasDraftGraphDiffVersusLive: false,
        hasDraftConsoleDiffVersusLive: false,
        hasDraftDiffVersusLive: false,
        hasCanvasStagingChanges: true,
        hasConsoleStagingChanges: false,
      }),
    ).toMatchObject({
      hasUncommittedCanvasDraftChanges: true,
      hasCommittedCanvasDraftChanges: false,
      hasUncommittedDraftChanges: true,
      hasCommittedDraftChanges: false,
      readyToPublishDraftChanges: false,
    });
  });

  it("shows only blue when branch differs from live with a clean staging area", () => {
    expect(
      getDraftChangeIndicators({
        suppressUnpublishedDraftDiscard: false,
        hasLatestDraftVersion: true,
        hasDraftGraphDiffVersusLive: false,
        hasDraftConsoleDiffVersusLive: true,
        hasDraftDiffVersusLive: true,
        hasCanvasStagingChanges: false,
        hasConsoleStagingChanges: false,
      }),
    ).toMatchObject({
      hasUncommittedConsoleDraftChanges: false,
      hasCommittedConsoleDraftChanges: true,
      hasUncommittedDraftChanges: false,
      hasCommittedDraftChanges: true,
      readyToPublishDraftChanges: true,
      readyToPublishConsoleDraftChanges: true,
    });
  });
});

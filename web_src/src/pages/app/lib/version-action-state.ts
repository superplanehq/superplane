type VersionActionAvailabilityInput = {
  hasEditableVersion: boolean;
  publishPending: boolean;
  canvasDeletedRemotely: boolean;
  isPreparingVersionAction: boolean;
  /** Live vs latest draft has node-level differences (same basis as draft discard UI). */
  hasDraftDiffVersusLive: boolean;
};

type VersionActionAvailability = {
  publishVersionDisabled: boolean;
  publishVersionDisabledTooltip?: string;
};

type DraftChangeIndicatorsInput = {
  suppressUnpublishedDraftDiscard: boolean;
  hasLatestDraftVersion: boolean;
  hasDraftGraphDiffVersusLive: boolean;
  hasDraftConsoleDiffVersusLive: boolean;
  hasDraftFilesDiffVersusLive: boolean;
  hasDraftDiffVersusLive: boolean;
};

type DraftChangeIndicators = {
  hasUnpublishedDraftChanges: boolean;
  hasUnpublishedCanvasDraftChanges: boolean;
  hasUnpublishedConsoleDraftChanges: boolean;
  hasUnpublishedFilesDraftChanges: boolean;
};

export function getDraftChangeIndicators({
  suppressUnpublishedDraftDiscard,
  hasLatestDraftVersion,
  hasDraftGraphDiffVersusLive,
  hasDraftConsoleDiffVersusLive,
  hasDraftFilesDiffVersusLive,
  hasDraftDiffVersusLive,
}: DraftChangeIndicatorsInput): DraftChangeIndicators {
  if (suppressUnpublishedDraftDiscard || !hasLatestDraftVersion) {
    return {
      hasUnpublishedDraftChanges: false,
      hasUnpublishedCanvasDraftChanges: false,
      hasUnpublishedConsoleDraftChanges: false,
      hasUnpublishedFilesDraftChanges: false,
    };
  }

  return {
    hasUnpublishedDraftChanges: hasDraftDiffVersusLive,
    hasUnpublishedCanvasDraftChanges: hasDraftGraphDiffVersusLive,
    hasUnpublishedConsoleDraftChanges: hasDraftConsoleDiffVersusLive,
    hasUnpublishedFilesDraftChanges: hasDraftFilesDiffVersusLive,
  };
}

function getPublishVersionDisabled({
  hasEditableVersion,
  publishPending,
  canvasDeletedRemotely,
  isPreparingVersionAction,
}: {
  hasEditableVersion: boolean;
  publishPending: boolean;
  canvasDeletedRemotely: boolean;
  isPreparingVersionAction: boolean;
}): boolean {
  return !hasEditableVersion || publishPending || canvasDeletedRemotely || isPreparingVersionAction;
}

function getPublishVersionDisabledTooltip({
  canvasDeletedRemotely,
  hasEditableVersion,
}: {
  canvasDeletedRemotely: boolean;
  hasEditableVersion: boolean;
}): string | undefined {
  if (canvasDeletedRemotely) {
    return "This canvas was deleted in another session.";
  }

  if (!hasEditableVersion) {
    return "Enable edit mode before publishing.";
  }

  return undefined;
}

export function getVersionActionAvailability({
  hasEditableVersion,
  publishPending,
  canvasDeletedRemotely,
  isPreparingVersionAction,
  hasDraftDiffVersusLive,
}: VersionActionAvailabilityInput): VersionActionAvailability {
  const publishVersionDisabledBase = getPublishVersionDisabled({
    hasEditableVersion,
    publishPending,
    canvasDeletedRemotely,
    isPreparingVersionAction,
  });

  const publishVersionDisabledTooltipBase = getPublishVersionDisabledTooltip({
    canvasDeletedRemotely,
    hasEditableVersion,
  });

  const publishVersionDisabled = publishVersionDisabledBase || !hasDraftDiffVersusLive;
  const publishVersionDisabledTooltip = publishVersionDisabledBase ? publishVersionDisabledTooltipBase : undefined;

  return {
    publishVersionDisabled,
    publishVersionDisabledTooltip,
  };
}

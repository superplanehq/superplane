type VersionActionAvailabilityInput = {
  hasEditableVersion: boolean;
  publishPending: boolean;
  canvasDeletedRemotely: boolean;
  isPreparingVersionAction: boolean;
  /** Live vs latest draft has node-level differences (same basis as draft discard UI). */
  hasDraftDiffVersusLive: boolean;
  /** Feature branch head differs from live main (e.g. README-only commits). */
  hasMergeableBranchChanges?: boolean;
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
  hasDraftDiffVersusLive: boolean;
};

type DraftChangeIndicators = {
  hasUnpublishedDraftChanges: boolean;
  hasUnpublishedCanvasDraftChanges: boolean;
  hasUnpublishedConsoleDraftChanges: boolean;
};

export function getDraftChangeIndicators({
  suppressUnpublishedDraftDiscard,
  hasLatestDraftVersion,
  hasDraftGraphDiffVersusLive,
  hasDraftConsoleDiffVersusLive,
  hasDraftDiffVersusLive,
}: DraftChangeIndicatorsInput): DraftChangeIndicators {
  if (suppressUnpublishedDraftDiscard || !hasLatestDraftVersion) {
    return {
      hasUnpublishedDraftChanges: false,
      hasUnpublishedCanvasDraftChanges: false,
      hasUnpublishedConsoleDraftChanges: false,
    };
  }

  return {
    hasUnpublishedDraftChanges: hasDraftDiffVersusLive,
    hasUnpublishedCanvasDraftChanges: hasDraftGraphDiffVersusLive,
    hasUnpublishedConsoleDraftChanges: hasDraftConsoleDiffVersusLive,
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

export function hasMergeableBranchChanges({
  isMainBranch,
  branchHeadVersionId,
  liveVersionId,
}: {
  isMainBranch: boolean;
  branchHeadVersionId?: string;
  liveVersionId?: string;
}): boolean {
  if (isMainBranch) {
    return false;
  }
  if (!branchHeadVersionId || !liveVersionId) {
    return false;
  }
  return branchHeadVersionId !== liveVersionId;
}

export function hasVersionActionChanges({
  hasDraftDiffVersusLive,
  hasMergeableBranchChanges = false,
}: {
  hasDraftDiffVersusLive: boolean;
  hasMergeableBranchChanges?: boolean;
}): boolean {
  return hasDraftDiffVersusLive || hasMergeableBranchChanges;
}

export function getVersionActionAvailability({
  hasEditableVersion,
  publishPending,
  canvasDeletedRemotely,
  isPreparingVersionAction,
  hasDraftDiffVersusLive,
  hasMergeableBranchChanges = false,
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

  const hasActionableChanges = hasVersionActionChanges({
    hasDraftDiffVersusLive,
    hasMergeableBranchChanges,
  });
  const publishVersionDisabled = publishVersionDisabledBase || !hasActionableChanges;
  const publishVersionDisabledTooltip = publishVersionDisabledBase ? publishVersionDisabledTooltipBase : undefined;

  return {
    publishVersionDisabled,
    publishVersionDisabledTooltip,
  };
}

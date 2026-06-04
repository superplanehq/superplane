type VersionActionAvailabilityInput = {
  isChangeManagementDisabled: boolean;
  hasEditableVersion: boolean;
  createChangeRequestPending: boolean;
  publishPending: boolean;
  canvasDeletedRemotely: boolean;
  isPreparingVersionAction: boolean;
  /** Live vs latest draft has node-level differences (same basis as draft discard UI). */
  hasDraftDiffVersusLive: boolean;
};

type VersionActionAvailability = {
  createChangeRequestDisabled: boolean;
  createChangeRequestDisabledTooltip?: string;
  publishVersionDisabled: boolean;
  publishVersionDisabledTooltip?: string;
};

type DraftChangeIndicatorsInput = {
  suppressUnpublishedDraftDiscard: boolean;
  hasLatestDraftVersion: boolean;
  hasDraftGraphDiffVersusLive: boolean;
  hasDraftConsoleDiffVersusLive: boolean;
  hasDraftDiffVersusLive: boolean;
  /** canvas.yaml staged content differs from the branch baseline. */
  hasCanvasStagingChanges?: boolean;
  /** console.yaml staged content differs from the branch baseline. */
  hasConsoleStagingChanges?: boolean;
  /** Other repository files staged in IndexedDB differ from the branch baseline. */
  hasFilesStagingChanges?: boolean;
  /** Repository files at branch HEAD differ from live (main). */
  hasDraftRepositoryFilesDiffVersusLive?: boolean;
};

export type DraftChangeIndicators = {
  /** Any uncommitted or committed draft change (legacy aggregate). */
  hasUnpublishedDraftChanges: boolean;
  hasUnpublishedCanvasDraftChanges: boolean;
  hasUnpublishedConsoleDraftChanges: boolean;
  hasUnpublishedFilesDraftChanges: boolean;
  /** IndexedDB staging differs from branch HEAD (orange). */
  hasUncommittedCanvasDraftChanges: boolean;
  hasUncommittedConsoleDraftChanges: boolean;
  hasUncommittedFilesDraftChanges: boolean;
  /** Branch HEAD / materialized draft differs from live (blue). */
  hasCommittedCanvasDraftChanges: boolean;
  hasCommittedConsoleDraftChanges: boolean;
  hasCommittedFilesDraftChanges: boolean;
  hasUncommittedDraftChanges: boolean;
  hasCommittedDraftChanges: boolean;
  /** Committed draft differs from live and staging is clean (blue UI only). */
  readyToPublishDraftChanges: boolean;
  readyToPublishCanvasDraftChanges: boolean;
  readyToPublishConsoleDraftChanges: boolean;
  readyToPublishFilesDraftChanges: boolean;
};

export function getDraftChangeIndicators({
  suppressUnpublishedDraftDiscard,
  hasLatestDraftVersion,
  hasDraftGraphDiffVersusLive,
  hasDraftConsoleDiffVersusLive,
  hasDraftRepositoryFilesDiffVersusLive,
  hasCanvasStagingChanges,
  hasConsoleStagingChanges,
  hasFilesStagingChanges,
}: DraftChangeIndicatorsInput): DraftChangeIndicators {
  if (suppressUnpublishedDraftDiscard || !hasLatestDraftVersion) {
    return {
      hasUnpublishedDraftChanges: false,
      hasUnpublishedCanvasDraftChanges: false,
      hasUnpublishedConsoleDraftChanges: false,
      hasUnpublishedFilesDraftChanges: false,
      hasUncommittedCanvasDraftChanges: false,
      hasUncommittedConsoleDraftChanges: false,
      hasUncommittedFilesDraftChanges: false,
      hasCommittedCanvasDraftChanges: false,
      hasCommittedConsoleDraftChanges: false,
      hasCommittedFilesDraftChanges: false,
      hasUncommittedDraftChanges: false,
      hasCommittedDraftChanges: false,
      readyToPublishDraftChanges: false,
      readyToPublishCanvasDraftChanges: false,
      readyToPublishConsoleDraftChanges: false,
      readyToPublishFilesDraftChanges: false,
    };
  }

  const hasUncommittedCanvasDraftChanges = !!hasCanvasStagingChanges;
  const hasUncommittedConsoleDraftChanges = !!hasConsoleStagingChanges;
  const hasUncommittedFilesDraftChanges = !!hasFilesStagingChanges;
  const hasCommittedCanvasDraftChanges = hasDraftGraphDiffVersusLive;
  const hasCommittedConsoleDraftChanges = hasDraftConsoleDiffVersusLive;
  const hasCommittedFilesDraftChanges = !!hasDraftRepositoryFilesDiffVersusLive;
  const hasUnpublishedCanvasDraftChanges = hasUncommittedCanvasDraftChanges || hasCommittedCanvasDraftChanges;
  const hasUnpublishedConsoleDraftChanges = hasUncommittedConsoleDraftChanges || hasCommittedConsoleDraftChanges;
  const hasUnpublishedFilesDraftChanges = hasUncommittedFilesDraftChanges || hasCommittedFilesDraftChanges;

  const hasUncommittedDraftChanges =
    hasUncommittedCanvasDraftChanges || hasUncommittedConsoleDraftChanges || hasUncommittedFilesDraftChanges;
  const hasCommittedDraftChanges =
    hasCommittedCanvasDraftChanges || hasCommittedConsoleDraftChanges || hasCommittedFilesDraftChanges;
  const readyToPublishDraftChanges = hasCommittedDraftChanges && !hasUncommittedDraftChanges;

  return {
    hasUnpublishedDraftChanges:
      hasUnpublishedCanvasDraftChanges || hasUnpublishedConsoleDraftChanges || hasUnpublishedFilesDraftChanges,
    hasUnpublishedCanvasDraftChanges,
    hasUnpublishedConsoleDraftChanges,
    hasUnpublishedFilesDraftChanges,
    hasUncommittedCanvasDraftChanges,
    hasUncommittedConsoleDraftChanges,
    hasUncommittedFilesDraftChanges,
    hasCommittedCanvasDraftChanges,
    hasCommittedConsoleDraftChanges,
    hasCommittedFilesDraftChanges,
    hasUncommittedDraftChanges,
    hasCommittedDraftChanges,
    readyToPublishDraftChanges,
    readyToPublishCanvasDraftChanges: hasCommittedCanvasDraftChanges && !hasUncommittedCanvasDraftChanges,
    readyToPublishConsoleDraftChanges: hasCommittedConsoleDraftChanges && !hasUncommittedConsoleDraftChanges,
    readyToPublishFilesDraftChanges: hasCommittedFilesDraftChanges && !hasUncommittedFilesDraftChanges,
  };
}

function getCreateChangeRequestDisabled({
  isChangeManagementDisabled,
  hasEditableVersion,
  createChangeRequestPending,
  canvasDeletedRemotely,
  isPreparingVersionAction,
}: {
  isChangeManagementDisabled: boolean;
  hasEditableVersion: boolean;
  createChangeRequestPending: boolean;
  canvasDeletedRemotely: boolean;
  isPreparingVersionAction: boolean;
}): boolean {
  return (
    isChangeManagementDisabled ||
    !hasEditableVersion ||
    createChangeRequestPending ||
    canvasDeletedRemotely ||
    isPreparingVersionAction
  );
}

function getCreateChangeRequestDisabledTooltip({
  canvasDeletedRemotely,
  isChangeManagementDisabled,
  hasEditableVersion,
}: {
  canvasDeletedRemotely: boolean;
  isChangeManagementDisabled: boolean;
  hasEditableVersion: boolean;
}): string | undefined {
  if (canvasDeletedRemotely) {
    return "This canvas was deleted in another session.";
  }

  if (isChangeManagementDisabled) {
    return "Change management is disabled for this canvas.";
  }

  if (!hasEditableVersion) {
    return "Enable edit mode before creating a change request.";
  }

  return undefined;
}

function getPublishVersionDisabled({
  isChangeManagementDisabled,
  hasEditableVersion,
  publishPending,
  canvasDeletedRemotely,
  isPreparingVersionAction,
  createChangeRequestDisabled,
}: {
  isChangeManagementDisabled: boolean;
  hasEditableVersion: boolean;
  publishPending: boolean;
  canvasDeletedRemotely: boolean;
  isPreparingVersionAction: boolean;
  createChangeRequestDisabled: boolean;
}): boolean {
  if (!isChangeManagementDisabled) {
    return createChangeRequestDisabled;
  }

  return !hasEditableVersion || publishPending || canvasDeletedRemotely || isPreparingVersionAction;
}

function getPublishVersionDisabledTooltip({
  isChangeManagementDisabled,
  canvasDeletedRemotely,
  hasEditableVersion,
  createChangeRequestDisabledTooltip,
}: {
  isChangeManagementDisabled: boolean;
  canvasDeletedRemotely: boolean;
  hasEditableVersion: boolean;
  createChangeRequestDisabledTooltip?: string;
}): string | undefined {
  if (!isChangeManagementDisabled) {
    return createChangeRequestDisabledTooltip;
  }

  if (canvasDeletedRemotely) {
    return "This canvas was deleted in another session.";
  }

  if (!hasEditableVersion) {
    return "Enable edit mode before publishing.";
  }

  return undefined;
}

export function getVersionActionAvailability({
  isChangeManagementDisabled,
  hasEditableVersion,
  createChangeRequestPending,
  publishPending,
  canvasDeletedRemotely,
  isPreparingVersionAction,
  hasDraftDiffVersusLive,
}: VersionActionAvailabilityInput): VersionActionAvailability {
  const createChangeRequestDisabled = getCreateChangeRequestDisabled({
    isChangeManagementDisabled,
    hasEditableVersion,
    createChangeRequestPending,
    canvasDeletedRemotely,
    isPreparingVersionAction,
  });

  const createChangeRequestDisabledTooltip = getCreateChangeRequestDisabledTooltip({
    canvasDeletedRemotely,
    isChangeManagementDisabled,
    hasEditableVersion,
  });

  const publishVersionDisabledBase = getPublishVersionDisabled({
    isChangeManagementDisabled,
    hasEditableVersion,
    publishPending,
    canvasDeletedRemotely,
    isPreparingVersionAction,
    createChangeRequestDisabled,
  });

  const publishVersionDisabledTooltipBase = getPublishVersionDisabledTooltip({
    isChangeManagementDisabled,
    canvasDeletedRemotely,
    hasEditableVersion,
    createChangeRequestDisabledTooltip,
  });

  const publishVersionDisabled = publishVersionDisabledBase || !hasDraftDiffVersusLive;
  const publishVersionDisabledTooltip = publishVersionDisabledBase ? publishVersionDisabledTooltipBase : undefined;

  return {
    createChangeRequestDisabled,
    createChangeRequestDisabledTooltip,
    publishVersionDisabled,
    publishVersionDisabledTooltip,
  };
}

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

import { useCallback, useRef, type MutableRefObject } from "react";

import { showErrorToast } from "@/lib/toast";

type UseEnterLiveEditSessionOptions = {
  organizationId?: string;
  canvasId?: string;
  canUpdateCanvas: boolean;
  effectiveLiveCanvasVersionId?: string;
  selectableVersionsById: Map<string, unknown>;
  handleUseVersion: (versionId: string, options?: { preserveStagedLayer?: boolean }) => void;
  resyncStagedEditorState: (
    versionId: string,
    options?: { bumpResetNonce?: boolean; preferCachedStaging?: boolean },
  ) => Promise<void>;
  previewingCurrentVersionRef: React.MutableRefObject<boolean>;
  setEditSessionActive: (value: boolean) => void;
  setIsEnteringEditSession: (value: boolean) => void;
};

export function useEnterLiveEditSession({
  organizationId,
  canvasId,
  canUpdateCanvas,
  effectiveLiveCanvasVersionId,
  selectableVersionsById,
  handleUseVersion,
  resyncStagedEditorState,
  previewingCurrentVersionRef,
  setEditSessionActive,
  setIsEnteringEditSession,
}: UseEnterLiveEditSessionOptions) {
  const enterLiveEditSessionInFlightRef = useRef<Promise<boolean> | null>(null);

  const enterLiveEditSession = useCallback(async (): Promise<boolean> => {
    if (enterLiveEditSessionInFlightRef.current) {
      return enterLiveEditSessionInFlightRef.current;
    }

    const enterPromise = (async (): Promise<boolean> => {
      if (!organizationId || !canvasId || !canUpdateCanvas) {
        return false;
      }

      if (!effectiveLiveCanvasVersionId) {
        return false;
      }

      if (!selectableVersionsById.has(effectiveLiveCanvasVersionId)) {
        return false;
      }

      setIsEnteringEditSession(true);
      try {
        previewingCurrentVersionRef.current = true;
        handleUseVersion(effectiveLiveCanvasVersionId, { preserveStagedLayer: true });
        await resyncStagedEditorState(effectiveLiveCanvasVersionId, {
          bumpResetNonce: false,
          preferCachedStaging: true,
        });
        setEditSessionActive(true);
        return true;
      } finally {
        setIsEnteringEditSession(false);
      }
    })();

    enterLiveEditSessionInFlightRef.current = enterPromise;
    try {
      return await enterPromise;
    } finally {
      enterLiveEditSessionInFlightRef.current = null;
    }
  }, [
    organizationId,
    canvasId,
    canUpdateCanvas,
    effectiveLiveCanvasVersionId,
    selectableVersionsById,
    handleUseVersion,
    resyncStagedEditorState,
    previewingCurrentVersionRef,
    setEditSessionActive,
    setIsEnteringEditSession,
  ]);

  return { enterLiveEditSession };
}

export async function returnToLiveVersionWhileEditing(options: {
  versionID: string;
  handleUseVersion: (versionID: string, options?: { preserveStagedLayer?: boolean }) => void;
  resyncStagedEditorState: (
    versionId: string,
    options?: { bumpResetNonce?: boolean; preferCachedStaging?: boolean },
  ) => Promise<void>;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
}): Promise<void> {
  options.previewingCurrentVersionRef.current = true;
  options.handleUseVersion(options.versionID, { preserveStagedLayer: true });
  await options.resyncStagedEditorState(options.versionID, {
    bumpResetNonce: false,
    preferCachedStaging: true,
  });
}

export async function navigateToCurrentVersion(options: {
  effectiveLiveCanvasVersionId?: string;
  editSessionActive: boolean;
  returnToLiveVersionWhileEditing: (versionID: string) => Promise<void>;
  handleUseVersion: (versionID: string) => void;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
}): Promise<void> {
  if (!options.effectiveLiveCanvasVersionId) {
    showErrorToast("No live version available");
    return;
  }

  if (options.editSessionActive) {
    await options.returnToLiveVersionWhileEditing(options.effectiveLiveCanvasVersionId);
    return;
  }

  options.previewingCurrentVersionRef.current = true;
  options.handleUseVersion(options.effectiveLiveCanvasVersionId);
}

type LiveVersionNavigationOptions = {
  effectiveLiveCanvasVersionId?: string;
  editSessionActive: boolean;
  handleUseVersion: (versionID: string, options?: { preserveStagedLayer?: boolean }) => void;
  resyncStagedEditorState: (
    versionId: string,
    options?: { bumpResetNonce?: boolean; preferCachedStaging?: boolean },
  ) => Promise<void>;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
};

export function useLiveVersionNavigation(options: LiveVersionNavigationOptions) {
  const returnToLive = useCallback(
    (versionID: string) =>
      returnToLiveVersionWhileEditing({
        versionID,
        handleUseVersion: options.handleUseVersion,
        resyncStagedEditorState: options.resyncStagedEditorState,
        previewingCurrentVersionRef: options.previewingCurrentVersionRef,
      }),
    [options.handleUseVersion, options.previewingCurrentVersionRef, options.resyncStagedEditorState],
  );

  const handleSeeCurrentVersion = useCallback(async () => {
    await navigateToCurrentVersion({
      effectiveLiveCanvasVersionId: options.effectiveLiveCanvasVersionId,
      editSessionActive: options.editSessionActive,
      returnToLiveVersionWhileEditing: returnToLive,
      handleUseVersion: options.handleUseVersion,
      previewingCurrentVersionRef: options.previewingCurrentVersionRef,
    });
  }, [
    options.editSessionActive,
    options.effectiveLiveCanvasVersionId,
    options.handleUseVersion,
    options.previewingCurrentVersionRef,
    returnToLive,
  ]);

  return { handleSeeCurrentVersion, returnToLiveVersionWhileEditing: returnToLive };
}

export function useVersionPanelSelection({
  hasEditableVersion,
  hasLocalSaveActivity,
  activeCanvasVersionIdRef,
  effectiveLiveCanvasVersionId,
  liveCanvasVersionId,
  editSessionActive,
  handleUseVersion,
  returnToLiveVersionWhileEditing,
  previewingCurrentVersionRef,
}: {
  hasEditableVersion: boolean;
  hasLocalSaveActivity: boolean;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  effectiveLiveCanvasVersionId?: string;
  liveCanvasVersionId?: string;
  editSessionActive: boolean;
  handleUseVersion: (versionID: string) => void;
  returnToLiveVersionWhileEditing: (versionID: string) => Promise<void>;
  previewingCurrentVersionRef: MutableRefObject<boolean>;
}) {
  return useCallback(
    async (versionID: string) => {
      if (hasEditableVersion && hasLocalSaveActivity && versionID !== activeCanvasVersionIdRef.current) {
        const shouldSwitch = window.confirm(
          "You have unsaved changes in the current draft. Switch versions and discard those unsaved changes?",
        );
        if (!shouldSwitch) {
          return;
        }
      }

      const isLiveVersion =
        (!!effectiveLiveCanvasVersionId && versionID === effectiveLiveCanvasVersionId) ||
        (!!liveCanvasVersionId && versionID === liveCanvasVersionId);
      previewingCurrentVersionRef.current = isLiveVersion;

      if (isLiveVersion && editSessionActive) {
        await returnToLiveVersionWhileEditing(versionID);
        return;
      }

      handleUseVersion(versionID);
    },
    [
      activeCanvasVersionIdRef,
      editSessionActive,
      effectiveLiveCanvasVersionId,
      handleUseVersion,
      hasEditableVersion,
      hasLocalSaveActivity,
      liveCanvasVersionId,
      previewingCurrentVersionRef,
      returnToLiveVersionWhileEditing,
    ],
  );
}

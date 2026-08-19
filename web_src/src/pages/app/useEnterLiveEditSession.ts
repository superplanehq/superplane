import { useCallback, useRef } from "react";

type UseEnterLiveEditSessionOptions = {
  organizationId?: string;
  canvasId?: string;
  canUpdateCanvas: boolean;
  liveCanvasVersionId?: string;
  selectableVersionsById: Map<string, unknown>;
  handleUseVersion: (versionId: string, options?: { preserveStagedLayer?: boolean }) => void;
  resyncStagedEditorState: (
    versionId: string,
    options?: { bumpResetNonce?: boolean; preferCachedStagedSpec?: boolean },
  ) => Promise<void>;
  previewingCurrentVersionRef: React.MutableRefObject<boolean>;
  setEditSessionActive: (value: boolean) => void;
  setIsEnteringEditSession: (value: boolean) => void;
};

export function useEnterLiveEditSession({
  organizationId,
  canvasId,
  canUpdateCanvas,
  liveCanvasVersionId,
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

      if (!liveCanvasVersionId) {
        return false;
      }

      if (!selectableVersionsById.has(liveCanvasVersionId)) {
        return false;
      }

      setIsEnteringEditSession(true);
      try {
        previewingCurrentVersionRef.current = true;
        handleUseVersion(liveCanvasVersionId, { preserveStagedLayer: true });
        await resyncStagedEditorState(liveCanvasVersionId, {
          bumpResetNonce: false,
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
    liveCanvasVersionId,
    selectableVersionsById,
    handleUseVersion,
    resyncStagedEditorState,
    previewingCurrentVersionRef,
    setEditSessionActive,
    setIsEnteringEditSession,
  ]);

  return { enterLiveEditSession };
}

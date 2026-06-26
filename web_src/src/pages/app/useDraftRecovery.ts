import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import type { useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { showErrorToast, showInfoToast, showSuccessToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";

import { recoverIfDraftMissing as resolveMissingDraftRecovery } from "./lib/draft-missing-recovery";
import { exitDraftToLive } from "./lib/exit-draft-to-live";
import { publishDraftVersionAndExit } from "./lib/publish-draft-flow";
import { VERSION_ACTION_TOAST_ID } from "./lib/version-action-toast";
import type { RefreshLatestLiveCanvasDataOptions } from "./useRefreshLatestLiveCanvasData";

type DraftSpec = CanvasesCanvas["spec"] | null;
type SetSearchParams = ReturnType<typeof useSearchParams>[1];
type PublishMutation = { mutateAsync: (versionId: string) => Promise<unknown> };

type UseDraftRecoveryOptions = {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionId: string;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  exitToLive: () => void;
  setSearchParams: SetSearchParams;
  refreshLatestLiveCanvasData: (options?: RefreshLatestLiveCanvasDataOptions) => Promise<void>;
  cancelPendingCanvasSaves?: () => void;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  publishCanvasVersionMutation: PublishMutation;
  setIsPreparingVersionAction: Dispatch<SetStateAction<boolean>>;
  registerIgnoredCanvasUpdatedEcho?: () => () => void;
  registerIgnoredCanvasVersionUpdatedEcho?: (versionId?: string) => () => void;
};

export function useDraftRecovery({
  organizationId,
  canvasId,
  activeCanvasVersionId,
  activeCanvasVersionIdRef,
  draftCanvasSpecsRef,
  setActiveCanvasVersion,
  setDraftCanvasSpec,
  exitToLive,
  setSearchParams,
  refreshLatestLiveCanvasData,
  cancelPendingCanvasSaves,
  ensureVersionActionDraftReady,
  publishCanvasVersionMutation,
  setIsPreparingVersionAction,
  registerIgnoredCanvasUpdatedEcho,
  registerIgnoredCanvasVersionUpdatedEcho,
}: UseDraftRecoveryOptions) {
  const queryClient = useQueryClient();

  const runExitDraftToLive = useCallback(
    (versionId: string, options?: RefreshLatestLiveCanvasDataOptions) =>
      exitDraftToLive({
        versionId,
        options,
        activeCanvasVersionIdRef,
        draftCanvasSpecsRef,
        setActiveCanvasVersion,
        setDraftCanvasSpec,
        canvasId,
        queryClient,
        exitToLive,
        setSearchParams,
        refreshLatestLiveCanvasData,
      }),
    [
      activeCanvasVersionIdRef,
      canvasId,
      draftCanvasSpecsRef,
      exitToLive,
      queryClient,
      refreshLatestLiveCanvasData,
      setActiveCanvasVersion,
      setDraftCanvasSpec,
      setSearchParams,
    ],
  );

  const recoverFromMissingDraft = useCallback(
    async (versionId: string, message = "This draft no longer exists. Returned to the live canvas.") => {
      cancelPendingCanvasSaves?.();
      await runExitDraftToLive(versionId);
      showInfoToast(message, { id: VERSION_ACTION_TOAST_ID });
    },
    [cancelPendingCanvasSaves, runExitDraftToLive],
  );

  const recoverIfDraftMissing = useCallback(
    (error: unknown, versionId: string) =>
      resolveMissingDraftRecovery({
        error,
        versionId,
        organizationId,
        canvasId,
        queryClient,
        recoverFromMissingDraft,
      }),
    [organizationId, canvasId, queryClient, recoverFromMissingDraft],
  );

  const handlePublishVersion = useCallback(async () => {
    if (!organizationId || !canvasId || !activeCanvasVersionId) {
      return;
    }

    setIsPreparingVersionAction(true);
    try {
      const result = await publishDraftVersionAndExit({
        organizationId,
        canvasId,
        activeCanvasVersionIdRef,
        queryClient,
        ensureVersionActionDraftReady,
        publishCanvasVersionMutation,
        registerIgnoredCanvasUpdatedEcho,
        registerIgnoredCanvasVersionUpdatedEcho,
        runExitDraftToLive,
        recoverFromMissingDraft,
      });
      if (result.status === "published") {
        showSuccessToast("Version published", { id: VERSION_ACTION_TOAST_ID });
        return;
      }
      if (result.status === "failed") {
        if (await recoverIfDraftMissing(result.error, result.versionIdToPublish)) {
          return;
        }
        showErrorToast(
          getUsageLimitToastMessage(result.error, getApiErrorMessage(result.error, "Failed to publish version")),
          { id: VERSION_ACTION_TOAST_ID },
        );
      }
    } finally {
      setIsPreparingVersionAction(false);
    }
  }, [
    organizationId,
    canvasId,
    activeCanvasVersionId,
    activeCanvasVersionIdRef,
    queryClient,
    ensureVersionActionDraftReady,
    publishCanvasVersionMutation,
    setIsPreparingVersionAction,
    registerIgnoredCanvasUpdatedEcho,
    registerIgnoredCanvasVersionUpdatedEcho,
    runExitDraftToLive,
    recoverFromMissingDraft,
    recoverIfDraftMissing,
  ]);

  return { handlePublishVersion, recoverFromMissingDraft, recoverIfDraftMissing };
}

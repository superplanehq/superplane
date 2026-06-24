import { useCallback, type Dispatch, type MutableRefObject, type SetStateAction } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { fetchCanvasVersionWithSpec } from "@/pages/app/lib/repository-spec-files";
import { syncCommittedCanvasDraftState } from "@/pages/app/lib/sync-committed-canvas-draft";

import { canvasKeys } from "./useCanvasData";

function isDraftVersionStillListed(queryClient: QueryClient, canvasId: string, versionId: string) {
  const branches = queryClient.getQueryData<CanvasesCanvasVersion[]>(canvasKeys.draftBranches(canvasId));
  if (!branches) {
    return true;
  }

  return branches.some((branch) => branch.metadata?.id === versionId);
}

type CanvasSpec = CanvasesCanvas["spec"] | null;

interface UseCanvasDraftResyncOptions {
  organizationId?: string;
  canvasId?: string;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, CanvasSpec>>;
  consoleMutationGenerationRef: MutableRefObject<number>;
  setDraftCanvasSpec: Dispatch<SetStateAction<CanvasSpec>>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setLastSavedWorkflowSnapshot: (workflow: CanvasesCanvas | null) => void;
  setStagingResetNonce: Dispatch<SetStateAction<number>>;
}

interface CanvasDraftResync {
  resyncDraftToCommitted: (versionId: string) => Promise<void>;
  resyncDraftToStaged: (versionId: string) => Promise<void>;
}

// Re-applies a draft version's effective spec (committed or staged) to the
// active editor state after a remote change. The rendered graph reads React
// state rather than a query cache, so only the actively-edited draft drives a
// full re-apply; other branches just drop their cached spec so a later switch
// refetches. Owned by a hook to keep this shared cross-tab sync logic out of
// the AppPage component body.
export function useCanvasDraftResync(options: UseCanvasDraftResyncOptions): CanvasDraftResync {
  const {
    organizationId,
    canvasId,
    activeCanvasVersionIdRef,
    draftCanvasSpecsRef,
    consoleMutationGenerationRef,
    setDraftCanvasSpec,
    setActiveCanvasVersion,
    setLastSavedWorkflowSnapshot,
    setStagingResetNonce,
  } = options;
  const queryClient = useQueryClient();

  // Applies an already-loaded spec to the active draft and treats it as the
  // saved baseline so it is not re-detected as a local edit (which would
  // re-stage and echo back to the originating tab, creating a feedback loop).
  const applyResyncedSpec = useCallback(
    (versionId: string, spec: CanvasSpec) => {
      if (!organizationId || !canvasId) {
        return;
      }

      if (spec) {
        draftCanvasSpecsRef.current.set(versionId, spec);
      } else {
        draftCanvasSpecsRef.current.delete(versionId);
      }
      setDraftCanvasSpec(spec);
      setActiveCanvasVersion((current) =>
        current?.metadata?.id === versionId ? { ...current, spec: spec ?? current.spec } : current,
      );

      const restored = queryClient.getQueryData<CanvasesCanvas>(canvasKeys.detail(organizationId, canvasId));
      if (restored) {
        setLastSavedWorkflowSnapshot(restored);
      }
    },
    [
      organizationId,
      canvasId,
      queryClient,
      draftCanvasSpecsRef,
      setDraftCanvasSpec,
      setActiveCanvasVersion,
      setLastSavedWorkflowSnapshot,
    ],
  );

  // A remote `canvas_version_updated` (e.g. a CLI `apps canvas update`) commits
  // canvas.yaml into the version row and discards the draft's staging. Refresh
  // the committed/staged caches and clear the staging indicators so the UI does
  // not keep showing stale "uncommitted changes" for content that no longer has
  // any staging on the server.
  const resyncDraftToCommitted = useCallback(
    async (versionId: string) => {
      if (!organizationId || !canvasId) {
        return;
      }

      if (!isDraftVersionStillListed(queryClient, canvasId, versionId)) {
        draftCanvasSpecsRef.current.delete(versionId);
        return;
      }

      await queryClient.invalidateQueries({ queryKey: canvasKeys.versionStaging(canvasId, versionId) });

      if (activeCanvasVersionIdRef.current !== versionId) {
        draftCanvasSpecsRef.current.delete(versionId);
        return;
      }

      consoleMutationGenerationRef.current += 1;
      const committedVersion = await syncCommittedCanvasDraftState({
        queryClient,
        organizationId,
        canvasId,
        versionId,
      });
      applyResyncedSpec(versionId, committedVersion?.spec ?? null);

      await queryClient.invalidateQueries({ queryKey: canvasKeys.console(canvasId, versionId) });
      setStagingResetNonce((nonce) => nonce + 1);
    },
    [
      organizationId,
      canvasId,
      queryClient,
      activeCanvasVersionIdRef,
      draftCanvasSpecsRef,
      consoleMutationGenerationRef,
      applyResyncedSpec,
      setStagingResetNonce,
    ],
  );

  // A remote `staging_updated` (another tab editing the same draft) changed the
  // draft's staging layer without committing. The console/files caches and diff
  // badge refresh through the websocket hook's invalidations, but the rendered
  // graph reads React state, so the active draft needs its effective staged
  // spec re-applied here.
  const resyncDraftToStaged = useCallback(
    async (versionId: string) => {
      if (!organizationId || !canvasId) {
        return;
      }

      if (!isDraftVersionStillListed(queryClient, canvasId, versionId)) {
        draftCanvasSpecsRef.current.delete(versionId);
        return;
      }

      await queryClient.invalidateQueries({ queryKey: canvasKeys.versionStaging(canvasId, versionId) });

      if (activeCanvasVersionIdRef.current !== versionId) {
        draftCanvasSpecsRef.current.delete(versionId);
        return;
      }

      consoleMutationGenerationRef.current += 1;
      const stagedVersion = await fetchCanvasVersionWithSpec(canvasId, versionId, true);
      const stagedSpec = stagedVersion?.spec ?? null;

      if (stagedVersion) {
        queryClient.setQueryData(canvasKeys.versionStagedDetail(canvasId, versionId), stagedVersion);
      }
      if (stagedSpec) {
        queryClient.setQueryData<CanvasesCanvas | undefined>(canvasKeys.detail(organizationId, canvasId), (current) =>
          current ? { ...current, spec: { ...current.spec, ...stagedSpec } } : current,
        );
      }
      applyResyncedSpec(versionId, stagedSpec);

      await queryClient.invalidateQueries({ queryKey: canvasKeys.consoleStaged(canvasId, versionId) });
      setStagingResetNonce((nonce) => nonce + 1);
    },
    [
      organizationId,
      canvasId,
      queryClient,
      activeCanvasVersionIdRef,
      draftCanvasSpecsRef,
      consoleMutationGenerationRef,
      applyResyncedSpec,
      setStagingResetNonce,
    ],
  );

  return { resyncDraftToCommitted, resyncDraftToStaged };
}

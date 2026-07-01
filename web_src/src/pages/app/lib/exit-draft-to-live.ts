import type { QueryClient } from "@tanstack/react-query";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import type { useSearchParams } from "react-router-dom";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { draftVersionId } from "@/lib/draftVersion";

import { clearPublishedDraftVersion } from "./draft-spec-cache";
import type { RefreshLatestLiveCanvasDataOptions } from "../useRefreshLatestLiveCanvasData";
import { clearComponentSidebarSearchParams } from "../viewState";

type DraftSpec = CanvasesCanvas["spec"] | null;
type SetSearchParams = ReturnType<typeof useSearchParams>[1];

export async function exitDraftToLive({
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
}: {
  versionId: string;
  options?: RefreshLatestLiveCanvasDataOptions;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  draftCanvasSpecsRef: MutableRefObject<Map<string, DraftSpec>>;
  setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>>;
  setDraftCanvasSpec: Dispatch<SetStateAction<DraftSpec>>;
  canvasId?: string;
  queryClient: QueryClient;
  exitToLive: () => void;
  setSearchParams: SetSearchParams;
  refreshLatestLiveCanvasData: (options?: RefreshLatestLiveCanvasDataOptions) => Promise<void>;
}) {
  activeCanvasVersionIdRef.current = "";
  if (versionId) {
    clearPublishedDraftVersion(draftCanvasSpecsRef.current, setActiveCanvasVersion, setDraftCanvasSpec, versionId);
  }
  if (canvasId && versionId && options?.skipDraftBranchRefetch) {
    queryClient.setQueryData<CanvasesCanvasVersion[]>(canvasKeys.draftBranches(canvasId), (current) =>
      (current ?? []).filter((branch) => draftVersionId(branch) !== versionId),
    );
  }
  exitToLive();
  setSearchParams((current) => {
    const next = new URLSearchParams(current);
    next.delete("version");
    next.delete("branch");
    return clearComponentSidebarSearchParams(next);
  });
  await refreshLatestLiveCanvasData(options);
}

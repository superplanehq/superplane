import { useMemo } from "react";

import type { CanvasesCanvasVersion } from "@/api-client";
import { useCanvasVersion } from "@/hooks/useCanvasData";

import { hasDraftVersusLiveGraphDiff } from "./draftNodeDiff";
import type { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";

type CommittedBaselines = ReturnType<typeof useCommittedDraftBaselines>;

type UseDraftVersionGraphDiffOptions = {
  organizationId: string;
  canvasId: string;
  isEditing: boolean;
  activeCanvasVersionId: string;
  liveCanvasVersionId: string;
  liveCanvasVersion: CanvasesCanvasVersion | undefined;
  draftVersionsFromBranches: CanvasesCanvasVersion[];
  selectedCanvasVersion: CanvasesCanvasVersion | null;
  latestDraftVersion: CanvasesCanvasVersion | undefined;
  committedBaselines: CommittedBaselines;
};

export function useDraftVersionGraphDiff({
  organizationId,
  canvasId,
  isEditing,
  activeCanvasVersionId,
  liveCanvasVersionId,
  liveCanvasVersion,
  draftVersionsFromBranches,
  selectedCanvasVersion,
  latestDraftVersion,
  committedBaselines,
}: UseDraftVersionGraphDiffOptions) {
  const { data: liveCanvasVersionWithYamlSpec } = useCanvasVersion(
    organizationId,
    canvasId,
    liveCanvasVersionId || "",
    !!organizationId && !!canvasId && !!liveCanvasVersionId,
    false,
  );
  const liveVersionForGraphDiff = liveCanvasVersionWithYamlSpec ?? liveCanvasVersion;
  const draftVersionForGraphDiff = useMemo(() => {
    const versionShell =
      draftVersionsFromBranches.find((draft) => draft.metadata?.id === activeCanvasVersionId) ??
      (selectedCanvasVersion?.metadata?.id === activeCanvasVersionId ? selectedCanvasVersion : undefined) ??
      latestDraftVersion;
    if (!versionShell) {
      return undefined;
    }

    if (isEditing && committedBaselines.ready && committedBaselines.canvasSpec) {
      return { ...versionShell, spec: committedBaselines.canvasSpec };
    }

    return versionShell;
  }, [
    activeCanvasVersionId,
    committedBaselines.canvasSpec,
    committedBaselines.ready,
    draftVersionsFromBranches,
    isEditing,
    latestDraftVersion,
    selectedCanvasVersion,
  ]);

  const hasDraftGraphDiffVersusLive = useMemo(
    () => hasDraftVersusLiveGraphDiff(liveVersionForGraphDiff, draftVersionForGraphDiff),
    [draftVersionForGraphDiff, liveVersionForGraphDiff],
  );

  return { draftVersionForGraphDiff, hasDraftGraphDiffVersusLive, liveVersionForGraphDiff };
}

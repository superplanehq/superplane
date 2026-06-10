import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";

import type { CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { resolveDraftBranchEditStatus, type DraftBranchEditStatus } from "./lib/draft-branch-edit-status";
import { fetchCanvasVersionStagingState } from "./lib/repository-spec-files";
import { draftVersionId } from "@/lib/draftVersion";

type UseDraftBranchesEditStatusOptions = {
  canvasId?: string;
  draftBranches: CanvasesCanvasVersion[];
  activeVersionId?: string;
  /** When true, `activeHasUncommittedChanges` drives the active draft row. */
  useLocalActiveStatus: boolean;
  activeHasUncommittedChanges: boolean;
  activeServerHasUncommittedChanges: boolean;
  activeHasPublishableChanges: boolean;
  publishableChangesByVersionId: Map<string, boolean>;
};

export function useDraftBranchesEditStatus({
  canvasId,
  draftBranches,
  activeVersionId,
  useLocalActiveStatus,
  activeHasUncommittedChanges,
  activeServerHasUncommittedChanges,
  activeHasPublishableChanges,
  publishableChangesByVersionId,
}: UseDraftBranchesEditStatusOptions): Map<string, DraftBranchEditStatus> {
  const inactiveDrafts = useMemo(
    () => draftBranches.filter((draft) => draftVersionId(draft) !== activeVersionId),
    [activeVersionId, draftBranches],
  );

  const inactiveStagingQueries = useQueries({
    queries: inactiveDrafts.map((draft) => {
      const versionId = draftVersionId(draft);
      return {
        queryKey: canvasKeys.versionStaging(canvasId ?? "", versionId ?? ""),
        queryFn: () => fetchCanvasVersionStagingState(canvasId!, versionId!),
        enabled: !!canvasId && !!versionId,
        staleTime: 15_000,
      };
    }),
  });

  return useMemo(() => {
    const statusByVersionId = new Map<string, DraftBranchEditStatus>();

    for (const draft of draftBranches) {
      const versionId = draftVersionId(draft);
      if (!versionId) {
        continue;
      }

      if (versionId === activeVersionId) {
        const hasUncommitted = useLocalActiveStatus ? activeHasUncommittedChanges : activeServerHasUncommittedChanges;
        statusByVersionId.set(versionId, resolveDraftBranchEditStatus(hasUncommitted, activeHasPublishableChanges));
        continue;
      }

      const inactiveIndex = inactiveDrafts.findIndex((item) => draftVersionId(item) === versionId);
      const hasUncommitted = inactiveStagingQueries[inactiveIndex]?.data?.hasStaging ?? false;
      const hasPublishable = publishableChangesByVersionId.get(versionId) ?? false;
      statusByVersionId.set(versionId, resolveDraftBranchEditStatus(hasUncommitted, hasPublishable));
    }

    return statusByVersionId;
  }, [
    activeHasPublishableChanges,
    activeHasUncommittedChanges,
    activeServerHasUncommittedChanges,
    activeVersionId,
    draftBranches,
    inactiveDrafts,
    inactiveStagingQueries,
    publishableChangesByVersionId,
    useLocalActiveStatus,
  ]);
}

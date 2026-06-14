import { useQueries, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

import type { CanvasesCanvasVersion } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

import { hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";
import { resolveDraftBranchEditStatus, type DraftBranchEditStatus } from "./lib/draft-branch-edit-status";
import { fetchCanvasVersionStagingSummary, fetchConsoleSpecFromRepository } from "./lib/repository-spec-files";
import { draftVersionId } from "@/lib/draftVersion";

type UseDraftBranchesEditStatusOptions = {
  canvasId?: string;
  draftBranches: CanvasesCanvasVersion[];
  activeVersionId?: string;
  liveVersionId?: string;
  /** When true, `activeHasUncommittedChanges` drives the active draft row. */
  useLocalActiveStatus: boolean;
  activeHasUncommittedChanges: boolean;
  activeServerHasUncommittedChanges: boolean;
  activeHasPublishableChanges: boolean;
  publishableChangesByVersionId: Map<string, boolean>;
};

// Reads the committed console of a version for the publishable diff. Kept as a
// prefix-extension of `canvasKeys.console` so a commit (which invalidates that
// key) also refreshes this entry. A dedicated suffix avoids colliding with the
// `CanvasConsoleData` shape that `useCanvasConsole` stores under the base key.
function publishableConsoleKey(canvasId: string, versionId: string) {
  return [...canvasKeys.console(canvasId, versionId), "publishable-diff"] as const;
}

export function useDraftBranchesEditStatus({
  canvasId,
  draftBranches,
  activeVersionId,
  liveVersionId,
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
        queryFn: () => fetchCanvasVersionStagingSummary(canvasId!, versionId!),
        enabled: !!canvasId && !!versionId,
        staleTime: 15_000,
      };
    }),
  });

  // The active draft's badge factors in console changes (via
  // `activeHasPublishableChanges`), so inactive drafts must too. The graph-only
  // `publishableChangesByVersionId` map misses drafts whose only change is the
  // console, so diff each inactive draft's committed console against live here.
  const liveConsoleQuery = useQuery({
    queryKey: publishableConsoleKey(canvasId ?? "", liveVersionId ?? ""),
    queryFn: () => fetchConsoleSpecFromRepository(canvasId!, liveVersionId!, false),
    enabled: !!canvasId && !!liveVersionId,
    staleTime: 15_000,
  });

  const inactiveConsoleQueries = useQueries({
    queries: inactiveDrafts.map((draft) => {
      const versionId = draftVersionId(draft);
      return {
        queryKey: publishableConsoleKey(canvasId ?? "", versionId ?? ""),
        queryFn: () => fetchConsoleSpecFromRepository(canvasId!, versionId!, false),
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
      const hasGraphPublishable = publishableChangesByVersionId.get(versionId) ?? false;
      const hasConsolePublishable = hasDraftVersusLiveConsoleDiff(
        liveConsoleQuery.data,
        inactiveConsoleQueries[inactiveIndex]?.data,
      );
      const hasPublishable = hasGraphPublishable || hasConsolePublishable;
      statusByVersionId.set(versionId, resolveDraftBranchEditStatus(hasUncommitted, hasPublishable));
    }

    return statusByVersionId;
  }, [
    activeHasPublishableChanges,
    activeHasUncommittedChanges,
    activeServerHasUncommittedChanges,
    activeVersionId,
    draftBranches,
    inactiveConsoleQueries,
    inactiveDrafts,
    inactiveStagingQueries,
    liveConsoleQuery.data,
    publishableChangesByVersionId,
    useLocalActiveStatus,
  ]);
}

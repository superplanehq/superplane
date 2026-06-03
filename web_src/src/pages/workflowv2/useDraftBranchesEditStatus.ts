import type { CanvasesCanvasDraftBranch, CanvasesCanvasDashboard, CanvasesCanvasVersion } from "@/api-client";
import type { DraftBranchEditStatus } from "@/components/CanvasToolSidebar/DraftBranchRow";
import { useEffect, useMemo, useState } from "react";
import {
  aggregateDraftTabIndicators,
  computeDraftBranchChangeDetail,
  detailToEditStatus,
  resolveActiveBranchChangeDetail,
  type DraftBranchChangeDetail,
} from "./lib/draft-branch-edit-status";
import type { DraftChangeIndicators } from "./lib/version-action-state";

type UseDraftBranchesEditStatusArgs = {
  canvasId: string | undefined;
  draftBranches: CanvasesCanvasDraftBranch[];
  liveCanvasVersion: CanvasesCanvasVersion | undefined;
  liveDashboard: CanvasesCanvasDashboard | null | undefined;
  activeBranch: string | null;
  activeBranchIndicatorsReady: boolean;
  activeBranchHasUncommittedCanvas: boolean;
  activeBranchHasUncommittedConsole: boolean;
  activeBranchHasUncommittedFiles: boolean;
  activeBranchHasCommittedCanvasVersusLive: boolean;
  activeBranchHasCommittedConsoleVersusLive: boolean;
  /** While true, hide transient "no-changes" on the active branch until the starter placeholder is staged. */
  pendingPlaceholderBoot?: boolean;
};

type UseDraftBranchesEditStatusResult = {
  draftBranchEditStatusByBranch: Record<string, DraftBranchEditStatus>;
  liveModeDraftChangeIndicators: DraftChangeIndicators | null;
};

export function useDraftBranchesEditStatus({
  canvasId,
  draftBranches,
  liveCanvasVersion,
  liveDashboard,
  activeBranch,
  activeBranchIndicatorsReady,
  activeBranchHasUncommittedCanvas,
  activeBranchHasUncommittedConsole,
  activeBranchHasUncommittedFiles,
  activeBranchHasCommittedCanvasVersusLive,
  activeBranchHasCommittedConsoleVersusLive,
  pendingPlaceholderBoot = false,
}: UseDraftBranchesEditStatusArgs): UseDraftBranchesEditStatusResult {
  const [detailsByBranch, setDetailsByBranch] = useState<Record<string, DraftBranchChangeDetail>>({});
  const activeDetail = useMemo(
    () =>
      activeBranch
        ? resolveActiveBranchChangeDetail(
            activeBranchHasUncommittedCanvas,
            activeBranchHasUncommittedConsole,
            activeBranchHasUncommittedFiles,
            activeBranchHasCommittedCanvasVersusLive,
            activeBranchHasCommittedConsoleVersusLive,
          )
        : undefined,
    [
      activeBranch,
      activeBranchHasUncommittedCanvas,
      activeBranchHasUncommittedConsole,
      activeBranchHasUncommittedFiles,
      activeBranchHasCommittedCanvasVersusLive,
      activeBranchHasCommittedConsoleVersusLive,
    ],
  );

  useEffect(() => {
    if (!canvasId || draftBranches.length === 0) {
      // Keep the same reference when already empty so an unstable `draftBranches`
      // identity (e.g. the `= []` default while the query loads) cannot trigger a
      // setState-on-every-render loop.
      setDetailsByBranch((prev) => (Object.keys(prev).length === 0 ? prev : {}));
      return;
    }

    let cancelled = false;

    void (async () => {
      const branches = draftBranches.filter((draft) => draft.branchName);

      const entries = await Promise.all(
        branches.map(async (draft) => {
          const branchName = draft.branchName!;
          const detail = await computeDraftBranchChangeDetail(
            canvasId,
            branchName,
            draft.tipSha,
            liveCanvasVersion,
            liveDashboard,
          );
          return [branchName, detail] as const;
        }),
      );

      if (cancelled) {
        return;
      }

      const next: Record<string, DraftBranchChangeDetail> = {};
      for (const [branchName, detail] of entries) {
        next[branchName] = detail;
      }
      setDetailsByBranch(next);
    })();

    return () => {
      cancelled = true;
    };
  }, [canvasId, draftBranches, liveCanvasVersion, liveDashboard]);

  const mergedDetailsByBranch = useMemo(() => {
    const merged = { ...detailsByBranch };
    // The active branch is driven by live staging indicators, not the async
    // IndexedDB scan (which can briefly report "no-changes" before staging loads).
    if (activeBranch) {
      delete merged[activeBranch];
    }
    if (activeBranch && activeBranchIndicatorsReady && activeDetail) {
      const hideNoChangesWhilePlaceholderPending = pendingPlaceholderBoot && activeDetail.editStatus === "no-changes";
      if (!hideNoChangesWhilePlaceholderPending) {
        merged[activeBranch] = activeDetail;
      }
    }
    return merged;
  }, [activeBranch, activeBranchIndicatorsReady, activeDetail, detailsByBranch, pendingPlaceholderBoot]);

  const draftBranchEditStatusByBranch = useMemo(() => {
    const statuses: Record<string, DraftBranchEditStatus> = {};
    for (const [branchName, detail] of Object.entries(mergedDetailsByBranch)) {
      statuses[branchName] = detailToEditStatus(detail);
    }
    return statuses;
  }, [mergedDetailsByBranch]);

  const liveModeDraftChangeIndicators = useMemo(() => {
    if (Object.keys(mergedDetailsByBranch).length === 0) {
      return null;
    }

    return aggregateDraftTabIndicators(mergedDetailsByBranch);
  }, [mergedDetailsByBranch]);

  return { draftBranchEditStatusByBranch, liveModeDraftChangeIndicators };
}

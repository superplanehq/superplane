import type { CanvasesCanvasVersion } from "@/api-client";
import { useCallback } from "react";
import type { CanvasVersionNodeDiffContext } from "@/pages/app/CanvasVersionNodeDiffDialog";
import type { DraftBranchEditStatus } from "@/pages/app/lib/draft-branch-edit-status";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { DraftBranchRow } from "./DraftBranchRow";
import { useVersionsTabScroll } from "./useVersionsTabScroll";
import { VersionRow } from "./VersionsTabPanelRow";

export interface VersionsTabPanelProps {
  scrollPersistenceKey?: string;
  liveCanvasVersionId?: string;
  liveCanvasVersion?: CanvasesCanvasVersion | null;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  liveVersions: CanvasesCanvasVersion[];
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onVersionNodeDiffContextChange: (context: CanvasVersionNodeDiffContext | null) => void;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
  draftBranches?: CanvasesCanvasVersion[];
  activeDraftBranch?: string | null;
  draftBranchEditStatusByVersionId?: Map<string, DraftBranchEditStatus>;
  onOpenDraftBranch?: (branchName: string) => void;
  onDeleteDraftBranch?: (versionId: string) => void;
  deleteDraftBranchPending?: boolean;
}

type VersionRowItem = {
  key: string;
  version: CanvasesCanvasVersion;
  isActive: boolean;
  isCurrentLive: boolean;
  isFirstCanvasVersion?: boolean;
  previousVersion?: CanvasesCanvasVersion;
  rowTestId?: string;
};

export function VersionsTabPanel({
  scrollPersistenceKey,
  liveCanvasVersionId,
  liveCanvasVersion,
  selectedCanvasVersion,
  liveVersions,
  canUpdateCanvas,
  canvasDeletedRemotely,
  onUseVersion,
  onVersionNodeDiffContextChange,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
  draftBranches,
  activeDraftBranch,
  draftBranchEditStatusByVersionId,
  onOpenDraftBranch,
  onDeleteDraftBranch,
  deleteDraftBranchPending,
}: VersionsTabPanelProps) {
  const { hasNoVersions, handleViewDiff, liveItems } = useVersionsPanelData({
    liveCanvasVersionId,
    liveCanvasVersion,
    selectedCanvasVersion,
    liveVersions,
    loadMoreLiveVersionsDisabled,
    onLoadMoreLiveVersions,
    onVersionNodeDiffContextChange,
  });
  const { scrollRef, handleScroll } = useVersionsTabScroll({
    scrollPersistenceKey,
    hasMore: Boolean(onLoadMoreLiveVersions) && !loadMoreLiveVersionsDisabled,
    isLoading: loadMoreLiveVersionsPending,
    onLoadMore: onLoadMoreLiveVersions,
    itemCount: liveItems.length,
  });

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-auto"
        data-testid="versions-sidebar-scroll"
        onScroll={handleScroll}
      >
        <VersionsNotices canUpdateCanvas={canUpdateCanvas} canvasDeletedRemotely={canvasDeletedRemotely} />

        <DraftBranchesSection
          drafts={draftBranches ?? []}
          activeDraftBranch={activeDraftBranch}
          draftBranchEditStatusByVersionId={draftBranchEditStatusByVersionId}
          canUpdateCanvas={canUpdateCanvas}
          deleteDraftBranchPending={deleteDraftBranchPending}
          onOpenDraftBranch={onOpenDraftBranch}
          onDeleteDraftBranch={onDeleteDraftBranch}
        />

        <section>
          {hasNoVersions ? (
            <p className="px-4 py-2 text-xs text-slate-600">No published history yet.</p>
          ) : (
            <VersionRowList items={liveItems} onUseVersion={onUseVersion} onViewDiff={handleViewDiff} />
          )}
        </section>
      </div>
    </div>
  );
}

function useVersionsPanelData({
  liveCanvasVersionId,
  liveCanvasVersion,
  selectedCanvasVersion,
  liveVersions,
  loadMoreLiveVersionsDisabled,
  onLoadMoreLiveVersions,
  onVersionNodeDiffContextChange,
}: Pick<
  VersionsTabPanelProps,
  | "liveCanvasVersionId"
  | "liveCanvasVersion"
  | "selectedCanvasVersion"
  | "liveVersions"
  | "loadMoreLiveVersionsDisabled"
  | "onLoadMoreLiveVersions"
  | "onVersionNodeDiffContextChange"
>) {
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "";
  const handleViewDiff = useCallback(
    (version: CanvasesCanvasVersion, previousVersion: CanvasesCanvasVersion) => {
      onVersionNodeDiffContextChange({ version, previousVersion });
    },
    [onVersionNodeDiffContextChange],
  );
  const liveItems = buildLiveItems({
    liveCanvasVersionId,
    liveVersions,
    loadMoreLiveVersionsDisabled,
    onLoadMoreLiveVersions,
    selectedVersionId,
  });
  const hasNoVersions = liveVersions.length === 0 && !liveCanvasVersion;

  return {
    hasNoVersions,
    handleViewDiff,
    liveItems,
  };
}

function DraftBranchesSection({
  drafts,
  activeDraftBranch,
  draftBranchEditStatusByVersionId,
  canUpdateCanvas,
  deleteDraftBranchPending,
  onOpenDraftBranch,
  onDeleteDraftBranch,
}: {
  drafts: CanvasesCanvasVersion[];
  activeDraftBranch?: string | null;
  draftBranchEditStatusByVersionId?: Map<string, DraftBranchEditStatus>;
  canUpdateCanvas: boolean;
  deleteDraftBranchPending?: boolean;
  onOpenDraftBranch?: (branchName: string) => void;
  onDeleteDraftBranch?: (versionId: string) => void;
}) {
  if (drafts.length === 0) {
    return (
      <section className="border-b border-slate-200 pb-2">
        <h3 className="px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500">Drafts</h3>
        <p className="px-4 pb-2 text-xs text-slate-600">No draft branches yet.</p>
      </section>
    );
  }

  return (
    <section className="border-b border-slate-200 pb-2" data-testid="canvas-drafts-section">
      <h3 className="px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500">Drafts</h3>
      {drafts.map((draft) => {
        const branchName = draftBranchName(draft);
        return (
          <DraftBranchRow
            key={branchName || draftVersionId(draft)}
            draft={draft}
            isActive={branchName === activeDraftBranch}
            editStatus={draftBranchEditStatusByVersionId?.get(draftVersionId(draft) ?? "") ?? "no-changes"}
            canUpdateCanvas={canUpdateCanvas}
            deletePending={deleteDraftBranchPending}
            onOpen={(nextBranchName) => onOpenDraftBranch?.(nextBranchName)}
            onDelete={onDeleteDraftBranch}
          />
        );
      })}
    </section>
  );
}

function VersionsNotices({
  canUpdateCanvas,
  canvasDeletedRemotely,
}: {
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
}) {
  return (
    <>
      {!canUpdateCanvas && !canvasDeletedRemotely ? (
        <p className="px-4 py-2 text-xs text-slate-600">You do not have permission to edit this canvas.</p>
      ) : null}
      {canvasDeletedRemotely ? (
        <p className="px-4 py-2 text-xs text-red-700">This canvas was deleted from another session.</p>
      ) : null}
    </>
  );
}

function VersionRowList({
  items,
  onUseVersion,
  onViewDiff,
}: {
  items: VersionRowItem[];
  onUseVersion: (versionID: string) => void;
  onViewDiff: (version: CanvasesCanvasVersion, previousVersion: CanvasesCanvasVersion) => void;
}) {
  return items.map((item) => (
    <VersionRow
      key={item.key}
      rowTestId={item.rowTestId}
      version={item.version}
      isActive={item.isActive}
      isCurrentLive={item.isCurrentLive}
      isFirstCanvasVersion={item.isFirstCanvasVersion}
      previousVersion={item.previousVersion}
      onUseVersion={onUseVersion}
      onViewDiff={onViewDiff}
    />
  ));
}

function buildLiveItems({
  liveCanvasVersionId,
  liveVersions,
  loadMoreLiveVersionsDisabled,
  onLoadMoreLiveVersions,
  selectedVersionId,
}: {
  liveCanvasVersionId?: string;
  liveVersions: CanvasesCanvasVersion[];
  loadMoreLiveVersionsDisabled?: boolean;
  onLoadMoreLiveVersions?: () => void;
  selectedVersionId: string;
}): VersionRowItem[] {
  return liveVersions.map((version, index) => {
    const versionID = version.metadata?.id || "";
    const isFirstCanvasVersion =
      index === liveVersions.length - 1 && (onLoadMoreLiveVersions ? !!loadMoreLiveVersionsDisabled : true);
    return {
      key: versionID,
      rowTestId: "canvas-live-version-row",
      version,
      isActive: versionID === selectedVersionId,
      isCurrentLive: liveCanvasVersionId === versionID,
      isFirstCanvasVersion,
      previousVersion: liveVersions[index + 1],
    };
  });
}

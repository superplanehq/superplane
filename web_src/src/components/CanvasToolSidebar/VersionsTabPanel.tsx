import type { CanvasChangeManagement, CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useCallback, useState } from "react";
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
  pendingApprovalVersions?: Array<{
    version: CanvasesCanvasVersion;
    changeRequest: CanvasesCanvasChangeRequest;
  }>;
  rejectedVersions?: Array<{
    version: CanvasesCanvasVersion;
    changeRequest: CanvasesCanvasChangeRequest;
  }>;
  liveVersions: CanvasesCanvasVersion[];
  liveVersionChangeRequestsByVersionId?: Map<string, CanvasesCanvasChangeRequest>;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onVersionNodeDiffContextChange: (context: CanvasVersionNodeDiffContext | null) => void;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
  changeRequestApprovalConfig?: CanvasChangeManagement;
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
  changeRequest?: CanvasesCanvasChangeRequest;
  variant?: "default" | "rejected";
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
  pendingApprovalVersions,
  rejectedVersions,
  liveVersions,
  liveVersionChangeRequestsByVersionId,
  canUpdateCanvas,
  canvasDeletedRemotely,
  onUseVersion,
  onVersionNodeDiffContextChange,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
  changeRequestApprovalConfig,
  draftBranches,
  activeDraftBranch,
  draftBranchEditStatusByVersionId,
  onOpenDraftBranch,
  onDeleteDraftBranch,
  deleteDraftBranchPending,
}: VersionsTabPanelProps) {
  const {
    hasNoVersions,
    handleViewDiff,
    liveItems,
    pendingItems,
    rejectedItems,
    rejectedList,
    rejectedVersionsExpanded,
    setRejectedVersionsExpanded,
  } = useVersionsPanelData({
    liveCanvasVersionId,
    liveCanvasVersion,
    selectedCanvasVersion,
    pendingApprovalVersions,
    rejectedVersions,
    liveVersions,
    liveVersionChangeRequestsByVersionId,
    loadMoreLiveVersionsDisabled,
    onLoadMoreLiveVersions,
    onVersionNodeDiffContextChange,
  });
  const { scrollRef, handleScroll } = useVersionsTabScroll({
    scrollPersistenceKey,
    hasMore: Boolean(onLoadMoreLiveVersions) && !loadMoreLiveVersionsDisabled,
    isLoading: loadMoreLiveVersionsPending,
    onLoadMore: onLoadMoreLiveVersions,
    itemCount: liveItems.length + pendingItems.length + rejectedItems.length,
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
            <VersionHistorySection
              items={[...pendingItems, ...liveItems]}
              changeRequestApprovalConfig={changeRequestApprovalConfig}
              onUseVersion={onUseVersion}
              onViewDiff={handleViewDiff}
            />
          )}
          <RejectedVersionsSection
            count={rejectedList.length}
            expanded={rejectedVersionsExpanded}
            items={rejectedItems}
            changeRequestApprovalConfig={changeRequestApprovalConfig}
            onToggleExpanded={() => setRejectedVersionsExpanded((value) => !value)}
            onUseVersion={onUseVersion}
            onViewDiff={handleViewDiff}
          />
        </section>
      </div>
    </div>
  );
}

function useVersionsPanelData({
  liveCanvasVersionId,
  liveCanvasVersion,
  selectedCanvasVersion,
  pendingApprovalVersions,
  rejectedVersions,
  liveVersions,
  liveVersionChangeRequestsByVersionId,
  loadMoreLiveVersionsDisabled,
  onLoadMoreLiveVersions,
  onVersionNodeDiffContextChange,
}: Pick<
  VersionsTabPanelProps,
  | "liveCanvasVersionId"
  | "liveCanvasVersion"
  | "selectedCanvasVersion"
  | "pendingApprovalVersions"
  | "rejectedVersions"
  | "liveVersions"
  | "liveVersionChangeRequestsByVersionId"
  | "loadMoreLiveVersionsDisabled"
  | "onLoadMoreLiveVersions"
  | "onVersionNodeDiffContextChange"
>) {
  const rejectedList = rejectedVersions ?? [];
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "";
  const [rejectedVersionsExpanded, setRejectedVersionsExpanded] = useState(false);
  const handleViewDiff = useCallback(
    (
      version: CanvasesCanvasVersion,
      previousVersion: CanvasesCanvasVersion,
      changeRequest?: CanvasesCanvasChangeRequest,
    ) => {
      onVersionNodeDiffContextChange({ version, previousVersion, changeRequest });
    },
    [onVersionNodeDiffContextChange],
  );
  const baselineVersion = liveVersions[0] ?? liveCanvasVersion ?? undefined;
  const pendingItems = buildPendingItems(pendingApprovalVersions ?? [], selectedVersionId, baselineVersion);
  const liveItems = buildLiveItems({
    liveCanvasVersionId,
    liveVersions,
    loadMoreLiveVersionsDisabled,
    liveVersionChangeRequestsByVersionId,
    onLoadMoreLiveVersions,
    selectedVersionId,
  });
  const rejectedItems = buildRejectedItems(rejectedList, selectedVersionId, baselineVersion);
  const hasNoVersions = liveVersions.length === 0 && pendingItems.length === 0 && rejectedItems.length === 0;

  return {
    hasNoVersions,
    handleViewDiff,
    liveItems,
    pendingItems,
    rejectedItems,
    rejectedList,
    rejectedVersionsExpanded,
    setRejectedVersionsExpanded,
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

function VersionHistorySection({
  items,
  changeRequestApprovalConfig,
  onUseVersion,
  onViewDiff,
}: {
  items: VersionRowItem[];
  changeRequestApprovalConfig?: CanvasChangeManagement;
  onUseVersion: (versionID: string) => void;
  onViewDiff: (
    version: CanvasesCanvasVersion,
    previousVersion: CanvasesCanvasVersion,
    changeRequest?: CanvasesCanvasChangeRequest,
  ) => void;
}) {
  return (
    <VersionRowList
      items={items}
      changeRequestApprovalConfig={changeRequestApprovalConfig}
      onUseVersion={onUseVersion}
      onViewDiff={onViewDiff}
    />
  );
}

function VersionRowList({
  items,
  changeRequestApprovalConfig,
  onUseVersion,
  onViewDiff,
}: {
  items: VersionRowItem[];
  changeRequestApprovalConfig?: CanvasChangeManagement;
  onUseVersion: (versionID: string) => void;
  onViewDiff: (
    version: CanvasesCanvasVersion,
    previousVersion: CanvasesCanvasVersion,
    changeRequest?: CanvasesCanvasChangeRequest,
  ) => void;
}) {
  return items.map((item) => (
    <VersionRow
      key={item.key}
      rowTestId={item.rowTestId}
      version={item.version}
      changeRequest={item.changeRequest}
      changeRequestApprovalConfig={changeRequestApprovalConfig}
      variant={item.variant}
      isActive={item.isActive}
      isCurrentLive={item.isCurrentLive}
      isFirstCanvasVersion={item.isFirstCanvasVersion}
      previousVersion={item.previousVersion}
      onUseVersion={onUseVersion}
      onViewDiff={onViewDiff}
    />
  ));
}

function RejectedVersionsSection({
  count,
  expanded,
  items,
  changeRequestApprovalConfig,
  onToggleExpanded,
  onUseVersion,
  onViewDiff,
}: {
  count: number;
  expanded: boolean;
  items: VersionRowItem[];
  changeRequestApprovalConfig?: CanvasChangeManagement;
  onToggleExpanded: () => void;
  onUseVersion: (versionID: string) => void;
  onViewDiff: (
    version: CanvasesCanvasVersion,
    previousVersion: CanvasesCanvasVersion,
    changeRequest?: CanvasesCanvasChangeRequest,
  ) => void;
}) {
  if (count === 0) return null;

  return (
    <div className="border-t border-slate-200">
      <button
        type="button"
        className="flex w-full items-center gap-1 px-4 py-2 text-left text-xs font-medium text-slate-500"
        onClick={onToggleExpanded}
        aria-expanded={expanded}
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 shrink-0" aria-hidden />
        ) : (
          <ChevronRight className="h-4 w-4 shrink-0" aria-hidden />
        )}
        <span>Rejected ({count})</span>
      </button>
      {expanded ? (
        <VersionRowList
          items={items}
          changeRequestApprovalConfig={changeRequestApprovalConfig}
          onUseVersion={onUseVersion}
          onViewDiff={onViewDiff}
        />
      ) : null}
    </div>
  );
}

function buildPendingItems(
  pendingApprovalVersions: Array<{ version: CanvasesCanvasVersion; changeRequest: CanvasesCanvasChangeRequest }>,
  selectedVersionId: string,
  previousVersion?: CanvasesCanvasVersion,
): VersionRowItem[] {
  return pendingApprovalVersions.map((item) => {
    const versionID = item.version.metadata?.id || "";
    return {
      key: `pending-${versionID || item.changeRequest.metadata?.id || "unknown"}`,
      rowTestId: "canvas-pending-change-request-version-row",
      version: item.version,
      changeRequest: item.changeRequest,
      isActive: versionID === selectedVersionId,
      isCurrentLive: false,
      previousVersion,
    };
  });
}

function buildLiveItems({
  liveCanvasVersionId,
  liveVersions,
  loadMoreLiveVersionsDisabled,
  liveVersionChangeRequestsByVersionId,
  onLoadMoreLiveVersions,
  selectedVersionId,
}: {
  liveCanvasVersionId?: string;
  liveVersions: CanvasesCanvasVersion[];
  loadMoreLiveVersionsDisabled?: boolean;
  liveVersionChangeRequestsByVersionId?: Map<string, CanvasesCanvasChangeRequest>;
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
      changeRequest: versionID ? liveVersionChangeRequestsByVersionId?.get(versionID) : undefined,
      isActive: versionID === selectedVersionId,
      isCurrentLive: liveCanvasVersionId === versionID,
      isFirstCanvasVersion,
      previousVersion: liveVersions[index + 1],
    };
  });
}

function buildRejectedItems(
  rejectedVersions: Array<{ version: CanvasesCanvasVersion; changeRequest: CanvasesCanvasChangeRequest }>,
  selectedVersionId: string,
  previousVersion?: CanvasesCanvasVersion,
): VersionRowItem[] {
  return rejectedVersions.map((item) => {
    const versionID = item.version.metadata?.id || "";
    return {
      key: `rejected-${versionID || item.changeRequest.metadata?.id || "unknown"}`,
      version: item.version,
      changeRequest: item.changeRequest,
      variant: "rejected",
      isActive: versionID === selectedVersionId,
      isCurrentLive: false,
      previousVersion,
    };
  });
}

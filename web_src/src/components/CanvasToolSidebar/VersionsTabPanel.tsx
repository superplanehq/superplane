import type { CanvasesCanvasVersion } from "@/api-client";
import { Plus } from "lucide-react";
import type { DraftBranchEditStatus } from "@/pages/app/lib/draft-branch-edit-status";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { ReactNode } from "react";
import { DraftBranchRow } from "./DraftBranchRow";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";
import { useVersionsTabScroll } from "./useVersionsTabScroll";
import { VersionRow } from "./VersionsTabPanelRow";

export interface VersionsTabPanelProps {
  scrollPersistenceKey?: string;
  liveCanvasVersionId?: string;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  liveVersions: CanvasesCanvasVersion[];
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
  draftBranches?: CanvasesCanvasVersion[];
  activeDraftBranch?: string | null;
  draftBranchEditStatusByVersionId?: Map<string, DraftBranchEditStatus>;
  onOpenDraftBranch?: (branchName: string) => void;
  onCreateDraftBranch?: () => void;
  createDraftBranchPending?: boolean;
  onDeleteDraftBranch?: (versionId: string) => void;
  deleteDraftBranchPending?: boolean;
}

type VersionRowItem = {
  key: string;
  version: CanvasesCanvasVersion;
  isActive: boolean;
  isCurrentLive: boolean;
  isFirstCanvasVersion?: boolean;
  rowTestId?: string;
};

export function VersionsTabPanel({
  scrollPersistenceKey,
  liveCanvasVersionId,
  selectedCanvasVersion,
  liveVersions,
  canUpdateCanvas,
  canvasDeletedRemotely,
  onUseVersion,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
  draftBranches,
  activeDraftBranch,
  draftBranchEditStatusByVersionId,
  onOpenDraftBranch,
  onCreateDraftBranch,
  createDraftBranchPending,
  onDeleteDraftBranch,
  deleteDraftBranchPending,
}: VersionsTabPanelProps) {
  const { hasNoVersions, liveItems } = useVersionsPanelData({
    liveCanvasVersionId,
    selectedCanvasVersion,
    liveVersions,
    loadMoreLiveVersionsDisabled,
    onLoadMoreLiveVersions,
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
          onCreateDraftBranch={onCreateDraftBranch}
          createDraftBranchPending={createDraftBranchPending}
          onDeleteDraftBranch={onDeleteDraftBranch}
        />

        <section>
          <VersionsSectionHeader label="History" />
          {hasNoVersions ? (
            <p className="px-3 py-2 text-xs text-slate-600">No published history yet.</p>
          ) : (
            <VersionRowList items={liveItems} onUseVersion={onUseVersion} />
          )}
        </section>
      </div>
    </div>
  );
}

function useVersionsPanelData({
  liveCanvasVersionId,
  selectedCanvasVersion,
  liveVersions,
  loadMoreLiveVersionsDisabled,
  onLoadMoreLiveVersions,
}: Pick<
  VersionsTabPanelProps,
  | "liveCanvasVersionId"
  | "selectedCanvasVersion"
  | "liveVersions"
  | "loadMoreLiveVersionsDisabled"
  | "onLoadMoreLiveVersions"
>) {
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "";
  const liveItems = buildLiveItems({
    liveCanvasVersionId,
    liveVersions,
    loadMoreLiveVersionsDisabled,
    onLoadMoreLiveVersions,
    selectedVersionId,
  });
  const hasNoVersions = liveVersions.length === 0 && !liveCanvasVersionId;

  return {
    hasNoVersions,
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
  onCreateDraftBranch,
  createDraftBranchPending,
  onDeleteDraftBranch,
}: {
  drafts: CanvasesCanvasVersion[];
  activeDraftBranch?: string | null;
  draftBranchEditStatusByVersionId?: Map<string, DraftBranchEditStatus>;
  canUpdateCanvas: boolean;
  deleteDraftBranchPending?: boolean;
  onOpenDraftBranch?: (branchName: string) => void;
  onCreateDraftBranch?: () => void;
  createDraftBranchPending?: boolean;
  onDeleteDraftBranch?: (versionId: string) => void;
}) {
  const header = (
    <DraftsSectionHeader
      canCreate={canUpdateCanvas && !!onCreateDraftBranch}
      createPending={createDraftBranchPending}
      onCreateDraftBranch={onCreateDraftBranch}
    />
  );

  if (drafts.length === 0) {
    return (
      <section>
        {header}
        <p className="px-3 py-2 text-xs text-slate-600">No draft branches yet.</p>
      </section>
    );
  }

  return (
    <section data-testid="canvas-drafts-section">
      {header}
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

function VersionsSectionHeader({ label, action }: { label: string; action?: ReactNode }) {
  return (
    <div className={cn(RUNS_SIDEBAR_ROW_CLASS, "justify-between pr-1.5")}>
      <span className="min-w-0 truncate text-[11px] font-medium uppercase tracking-wide text-gray-500">{label}</span>
      {action}
    </div>
  );
}

function DraftsSectionHeader({
  canCreate,
  createPending,
  onCreateDraftBranch,
}: {
  canCreate: boolean;
  createPending?: boolean;
  onCreateDraftBranch?: () => void;
}) {
  return (
    <VersionsSectionHeader
      label="Drafts"
      action={
        canCreate ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                onClick={() => onCreateDraftBranch?.()}
                disabled={createPending}
                className="size-6 shrink-0 rounded p-0 text-gray-500 hover:bg-gray-100 hover:text-gray-700"
                data-testid="canvas-create-draft-button"
                aria-label="Create draft"
              >
                <Plus className="size-4" aria-hidden />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="top">Create new draft</TooltipContent>
          </Tooltip>
        ) : null
      }
    />
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
        <p className="px-3 py-2 text-xs text-slate-600">You do not have permission to edit this canvas.</p>
      ) : null}
      {canvasDeletedRemotely ? (
        <p className="px-3 py-2 text-xs text-red-700">This canvas was deleted from another session.</p>
      ) : null}
    </>
  );
}

function VersionRowList({
  items,
  onUseVersion,
}: {
  items: VersionRowItem[];
  onUseVersion: (versionID: string) => void;
}) {
  return items.map((item) => (
    <VersionRow
      key={item.key}
      rowTestId={item.rowTestId}
      version={item.version}
      isActive={item.isActive}
      isCurrentLive={item.isCurrentLive}
      isFirstCanvasVersion={item.isFirstCanvasVersion}
      onUseVersion={onUseVersion}
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
    };
  });
}

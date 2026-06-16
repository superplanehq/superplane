import type { CanvasesCanvasVersion } from "@/api-client";
import { useCallback } from "react";
import { Copy, GitBranch, Plus } from "lucide-react";
import { toast } from "sonner";
import type { CanvasVersionNodeDiffContext } from "@/pages/app/CanvasVersionNodeDiffDialog";
import type { DraftBranchEditStatus } from "@/pages/app/lib/draft-branch-edit-status";
import { draftBranchName, draftVersionId } from "@/lib/draftVersion";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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
  onCreateDraftBranch,
  createDraftBranchPending,
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
          onCreateDraftBranch={onCreateDraftBranch}
          createDraftBranchPending={createDraftBranchPending}
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

      <VersionsFooter />
    </div>
  );
}

// Placeholder until the canvas repository clone URL is wired through the API.
const PLACEHOLDER_CLONE_COMMAND = "git clone <canvas-repository-url>";

function VersionsFooter() {
  const handleCopyCloneCommand = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(PLACEHOLDER_CLONE_COMMAND);
      toast.success("Clone command copied");
    } catch {
      toast.error("Failed to copy clone command");
    }
  }, []);

  return (
    <div className="shrink-0 border-t border-slate-200 px-4 py-3" data-testid="versions-sidebar-footer">
      <div className="flex items-center gap-1.5 text-xs font-medium text-slate-600">
        <GitBranch className="size-3.5 text-slate-500" aria-hidden />
        <span>This canvas is git-backed</span>
      </div>
      <button
        type="button"
        onClick={handleCopyCloneCommand}
        className="mt-2 flex w-full items-center justify-between gap-2 rounded border border-slate-200 bg-slate-50 px-2 py-1.5 text-left text-xs text-slate-700 hover:bg-slate-100"
        data-testid="versions-sidebar-copy-clone-command"
        title="Copy clone command"
      >
        <code className="min-w-0 flex-1 truncate font-mono text-[11px]">{PLACEHOLDER_CLONE_COMMAND}</code>
        <Copy className="size-3.5 shrink-0 text-slate-500" aria-hidden />
      </button>
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
      <section className="border-b border-slate-200 pb-2">
        {header}
        <p className="px-4 pb-2 text-xs text-slate-600">No draft branches yet.</p>
      </section>
    );
  }

  return (
    <section className="border-b border-slate-200 pb-2" data-testid="canvas-drafts-section">
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
    <div className="flex items-center justify-between px-4 py-2">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500">Drafts</h3>
      {canCreate ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              onClick={() => onCreateDraftBranch?.()}
              disabled={createPending}
              className="flex size-5 items-center justify-center rounded text-slate-500 hover:bg-slate-100 hover:text-slate-700 disabled:cursor-not-allowed disabled:opacity-50"
              data-testid="canvas-create-draft-button"
              aria-label="Create draft"
            >
              <Plus className="size-4" aria-hidden />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top">Create new draft</TooltipContent>
        </Tooltip>
      ) : null}
    </div>
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

import type { CanvasesCanvasBranch, CanvasesCanvasVersion } from "@/api-client";
import { cn } from "@/lib/utils";
import { CanvasBranchSelector } from "@/ui/CanvasPage/components/CanvasBranchSelector";
import { useOrganizationUserAvatars } from "@/hooks/useOrganizationUserAvatars";
import { userRefDisplayProfile, type UserDisplayProfile } from "@/lib/userRefDisplay";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";
import { useVersionsTabScroll } from "./useVersionsTabScroll";
import { VersionRow } from "./VersionsTabPanelRow";

export interface VersionsTabPanelProps {
  organizationId?: string;
  scrollPersistenceKey?: string;
  branchHeadVersionId?: string;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  branchCommits: CanvasesCanvasVersion[];
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onLoadMoreBranchCommits?: () => void;
  loadMoreBranchCommitsDisabled?: boolean;
  loadMoreBranchCommitsPending?: boolean;
  canvasBranches?: CanvasesCanvasBranch[];
  activeBranchName?: string;
  onSelectBranch?: (branchName: string) => void;
  branchSelectorDisabled?: boolean;
}

type CommitRowItem = {
  key: string;
  version: CanvasesCanvasVersion;
  isActive: boolean;
  rowTestId?: string;
};

export function VersionsTabPanel({
  organizationId,
  scrollPersistenceKey,
  branchHeadVersionId,
  selectedCanvasVersion,
  branchCommits,
  canUpdateCanvas,
  canvasDeletedRemotely,
  onUseVersion,
  onLoadMoreBranchCommits,
  loadMoreBranchCommitsDisabled,
  loadMoreBranchCommitsPending,
  canvasBranches = [],
  activeBranchName,
  onSelectBranch,
  branchSelectorDisabled,
}: VersionsTabPanelProps) {
  const userAvatars = useOrganizationUserAvatars(organizationId);
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || branchHeadVersionId || "";
  const commitItems = buildCommitItems({
    branchCommits,
    selectedVersionId,
  });
  const { scrollRef, handleScroll } = useVersionsTabScroll({
    scrollPersistenceKey,
    hasMore: Boolean(onLoadMoreBranchCommits) && !loadMoreBranchCommitsDisabled,
    isLoading: loadMoreBranchCommitsPending,
    onLoadMore: onLoadMoreBranchCommits,
    itemCount: commitItems.length,
  });

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="border-b border-slate-200 px-3 py-2">
        {onSelectBranch && activeBranchName ? (
          <CanvasBranchSelector
            branches={canvasBranches}
            value={activeBranchName}
            onValueChange={onSelectBranch}
            disabled={branchSelectorDisabled || !canUpdateCanvas}
          />
        ) : null}
      </div>

      <div
        ref={scrollRef}
        className="min-h-0 flex-1 overflow-auto"
        data-testid="versions-sidebar-scroll"
        onScroll={handleScroll}
      >
        <VersionsNotices canUpdateCanvas={canUpdateCanvas} canvasDeletedRemotely={canvasDeletedRemotely} />

        <section>
          <VersionsSectionHeader label="Commits" />
          {commitItems.length === 0 ? (
            <p className="px-3 py-2 text-xs text-slate-600">No commits on this branch yet.</p>
          ) : (
            <VersionRowList items={commitItems} onUseVersion={onUseVersion} userAvatars={userAvatars} />
          )}
        </section>
      </div>
    </div>
  );
}

function VersionsSectionHeader({ label }: { label: string }) {
  return (
    <div className={cn(RUNS_SIDEBAR_ROW_CLASS, "justify-between pr-1.5")}>
      <span className="min-w-0 truncate text-[11px] font-medium uppercase tracking-wide text-gray-500">{label}</span>
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
  userAvatars,
}: {
  items: CommitRowItem[];
  onUseVersion: (versionID: string) => void;
  userAvatars: Map<string, UserDisplayProfile>;
}) {
  return items.map((item) => (
    <VersionRow
      key={item.key}
      rowTestId={item.rowTestId}
      version={item.version}
      isActive={item.isActive}
      committer={userRefDisplayProfile(item.version.metadata?.owner, userAvatars)}
      onUseVersion={onUseVersion}
    />
  ));
}

function buildCommitItems({
  branchCommits,
  selectedVersionId,
}: {
  branchCommits: CanvasesCanvasVersion[];
  selectedVersionId: string;
}): CommitRowItem[] {
  return branchCommits.map((version) => {
    const versionID = version.metadata?.id || "";
    return {
      key: versionID,
      rowTestId: "canvas-commit-row",
      version,
      isActive: versionID === selectedVersionId,
    };
  });
}

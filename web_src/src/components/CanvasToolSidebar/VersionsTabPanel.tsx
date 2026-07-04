import type { CanvasesCanvasVersion } from "@/api-client";
import { cn } from "@/lib/utils";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";
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
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
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
  liveCanvasVersion,
  selectedCanvasVersion,
  liveVersions,
  canUpdateCanvas,
  canvasDeletedRemotely,
  onUseVersion,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
}: VersionsTabPanelProps) {
  const { hasNoVersions, liveItems } = useVersionsPanelData({
    liveCanvasVersionId,
    liveCanvasVersion,
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

        <section>
          <VersionsSectionHeader label="History" />
          {hasNoVersions ? (
            <p className="px-3 py-2 text-xs text-slate-600">No commit history yet.</p>
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
  liveCanvasVersion,
  selectedCanvasVersion,
  liveVersions,
  loadMoreLiveVersionsDisabled,
  onLoadMoreLiveVersions,
}: Pick<
  VersionsTabPanelProps,
  | "liveCanvasVersionId"
  | "liveCanvasVersion"
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
  const hasNoVersions = liveVersions.length === 0 && !liveCanvasVersion;

  return {
    hasNoVersions,
    liveItems,
  };
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

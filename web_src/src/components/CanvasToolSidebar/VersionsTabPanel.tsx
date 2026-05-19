import type { CanvasChangeManagement, CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { ChevronDown, ChevronRight, GitBranch } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import type { CanvasVersionNodeDiffContext } from "@/pages/workflowv2/CanvasVersionNodeDiffDialog";
import { VersionRow } from "./VersionsTabPanelRow";

const INITIAL_VISIBLE_LIVE_VERSIONS = 5;
const LOAD_OLDER_LIVE_VERSIONS_STEP = 5;

export interface VersionsTabPanelProps {
  liveCanvasVersionId?: string;
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
  isTemplate: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onVersionNodeDiffContextChange: (context: CanvasVersionNodeDiffContext | null) => void;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
  changeRequestApprovalConfig?: CanvasChangeManagement;
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
  liveCanvasVersionId,
  selectedCanvasVersion,
  pendingApprovalVersions,
  rejectedVersions,
  liveVersions,
  liveVersionChangeRequestsByVersionId,
  canUpdateCanvas,
  isTemplate,
  canvasDeletedRemotely,
  onUseVersion,
  onVersionNodeDiffContextChange,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
  changeRequestApprovalConfig,
}: VersionsTabPanelProps) {
  const {
    hasNoVersions,
    handleLoadOlderVersions,
    handleViewDiff,
    liveItems,
    loadOlderVersionsDisabled,
    loadOlderVersionsPending,
    pendingItems,
    rejectedItems,
    rejectedList,
    rejectedVersionsExpanded,
    setRejectedVersionsExpanded,
    showLoadOlderVersions,
  } = useVersionsPanelData({
    liveCanvasVersionId,
    selectedCanvasVersion,
    pendingApprovalVersions,
    rejectedVersions,
    liveVersions,
    liveVersionChangeRequestsByVersionId,
    loadMoreLiveVersionsDisabled,
    loadMoreLiveVersionsPending,
    onLoadMoreLiveVersions,
    onVersionNodeDiffContextChange,
  });

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex h-10 shrink-0 items-center border-b border-slate-200 px-3">
        <span className="inline-flex items-center gap-2 text-sm font-medium text-slate-900">
          <GitBranch className="h-4 w-4" />
          Versions
        </span>
      </div>

      <div className="min-h-0 flex-1 overflow-auto p-3">
        <VersionsNotices
          canUpdateCanvas={canUpdateCanvas}
          canvasDeletedRemotely={canvasDeletedRemotely}
          isTemplate={isTemplate}
        />

        <section className="mt-3 rounded-md">
          {hasNoVersions ? (
            <p className="mt-2 text-xs text-slate-600">No published history yet.</p>
          ) : (
            <VersionHistorySection
              items={[...pendingItems, ...liveItems]}
              changeRequestApprovalConfig={changeRequestApprovalConfig}
              showLoadOlderVersions={showLoadOlderVersions}
              loadOlderVersionsPending={loadOlderVersionsPending}
              loadOlderVersionsDisabled={loadOlderVersionsDisabled}
              onLoadOlderVersions={handleLoadOlderVersions}
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
  selectedCanvasVersion,
  pendingApprovalVersions,
  rejectedVersions,
  liveVersions,
  liveVersionChangeRequestsByVersionId,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
  onLoadMoreLiveVersions,
  onVersionNodeDiffContextChange,
}: Pick<
  VersionsTabPanelProps,
  | "liveCanvasVersionId"
  | "selectedCanvasVersion"
  | "pendingApprovalVersions"
  | "rejectedVersions"
  | "liveVersions"
  | "liveVersionChangeRequestsByVersionId"
  | "loadMoreLiveVersionsDisabled"
  | "loadMoreLiveVersionsPending"
  | "onLoadMoreLiveVersions"
  | "onVersionNodeDiffContextChange"
>) {
  const rejectedList = rejectedVersions ?? [];
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "";
  const [rejectedVersionsExpanded, setRejectedVersionsExpanded] = useState(false);
  const {
    displayedLiveVersions,
    handleLoadOlderVersions,
    loadOlderVersionsDisabled,
    loadOlderVersionsPending,
    showLoadOlderVersions,
  } = useVisibleLiveVersions({
    liveVersions,
    loadMoreLiveVersionsDisabled,
    loadMoreLiveVersionsPending,
    onLoadMoreLiveVersions,
  });
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
  const pendingItems = buildPendingItems(pendingApprovalVersions ?? [], selectedVersionId, liveVersions[0]);
  const liveItems = buildLiveItems({
    displayedLiveVersions,
    liveCanvasVersionId,
    liveVersions,
    loadMoreLiveVersionsDisabled,
    liveVersionChangeRequestsByVersionId,
    onLoadMoreLiveVersions,
    selectedVersionId,
  });
  const rejectedItems = buildRejectedItems(rejectedList, selectedVersionId, liveVersions[0]);
  const hasNoVersions = liveVersions.length === 0 && pendingItems.length === 0 && rejectedItems.length === 0;

  return {
    hasNoVersions,
    handleLoadOlderVersions,
    handleViewDiff,
    liveItems,
    loadOlderVersionsDisabled,
    loadOlderVersionsPending,
    pendingItems,
    rejectedItems,
    rejectedList,
    rejectedVersionsExpanded,
    setRejectedVersionsExpanded,
    showLoadOlderVersions,
  };
}

function useVisibleLiveVersions({
  liveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
  onLoadMoreLiveVersions,
}: Pick<
  VersionsTabPanelProps,
  "liveVersions" | "loadMoreLiveVersionsDisabled" | "loadMoreLiveVersionsPending" | "onLoadMoreLiveVersions"
>) {
  const [visibleLiveVersionCount, setVisibleLiveVersionCount] = useState(INITIAL_VISIBLE_LIVE_VERSIONS);
  const headLiveVersionId = liveVersions[0]?.metadata?.id ?? "";

  useEffect(() => {
    setVisibleLiveVersionCount(INITIAL_VISIBLE_LIVE_VERSIONS);
  }, [headLiveVersionId]);

  const displayedLiveVersions = liveVersions.slice(0, visibleLiveVersionCount);
  const canExpandLocal = visibleLiveVersionCount < liveVersions.length;
  const showLoadOlderVersions = canExpandLocal || !!onLoadMoreLiveVersions;

  const handleLoadOlderVersions = useCallback(() => {
    if (canExpandLocal) {
      setVisibleLiveVersionCount((prev) => Math.min(prev + LOAD_OLDER_LIVE_VERSIONS_STEP, liveVersions.length));
      return;
    }
    onLoadMoreLiveVersions?.();
  }, [canExpandLocal, liveVersions.length, onLoadMoreLiveVersions]);

  return {
    displayedLiveVersions,
    handleLoadOlderVersions,
    loadOlderVersionsDisabled: !canExpandLocal && (loadMoreLiveVersionsDisabled ?? !onLoadMoreLiveVersions),
    loadOlderVersionsPending: !canExpandLocal && !!loadMoreLiveVersionsPending,
    showLoadOlderVersions,
  };
}

function VersionsNotices({
  canUpdateCanvas,
  canvasDeletedRemotely,
  isTemplate,
}: {
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  isTemplate: boolean;
}) {
  return (
    <>
      {!canUpdateCanvas && !canvasDeletedRemotely ? (
        <p className="text-xs text-slate-600">You do not have permission to edit this canvas.</p>
      ) : null}
      {canvasDeletedRemotely ? (
        <p className="text-xs text-red-700">This canvas was deleted from another session.</p>
      ) : null}
      {isTemplate ? <p className="text-xs text-slate-600">Template canvases are read-only.</p> : null}
    </>
  );
}

function VersionHistorySection({
  items,
  changeRequestApprovalConfig,
  showLoadOlderVersions,
  loadOlderVersionsPending,
  loadOlderVersionsDisabled,
  onLoadOlderVersions,
  onUseVersion,
  onViewDiff,
}: {
  items: VersionRowItem[];
  changeRequestApprovalConfig?: CanvasChangeManagement;
  showLoadOlderVersions: boolean;
  loadOlderVersionsPending: boolean;
  loadOlderVersionsDisabled: boolean;
  onLoadOlderVersions: () => void;
  onUseVersion: (versionID: string) => void;
  onViewDiff: (
    version: CanvasesCanvasVersion,
    previousVersion: CanvasesCanvasVersion,
    changeRequest?: CanvasesCanvasChangeRequest,
  ) => void;
}) {
  return (
    <>
      <div className="-mt-4 space-y-1">
        <VersionRowList
          items={items}
          changeRequestApprovalConfig={changeRequestApprovalConfig}
          onUseVersion={onUseVersion}
          onViewDiff={onViewDiff}
        />
      </div>
      <LoadOlderVersionsButton
        show={showLoadOlderVersions}
        pending={loadOlderVersionsPending}
        disabled={loadOlderVersionsDisabled}
        onClick={onLoadOlderVersions}
      />
    </>
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

function LoadOlderVersionsButton({
  show,
  pending,
  disabled,
  onClick,
}: {
  show: boolean;
  pending: boolean;
  disabled: boolean;
  onClick: () => void;
}) {
  if (!show) return null;

  return (
    <Button variant="outline" size="sm" className="mt-2 w-fit self-start" onClick={onClick} disabled={disabled}>
      {pending ? "Loading..." : "Load older versions"}
    </Button>
  );
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
    <div className="mt-3 border-t border-slate-200 pt-3">
      <button
        type="button"
        className="flex w-full items-center gap-1 rounded-md py-1.5 text-left text-xs font-medium text-slate-500"
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
        <div className="mt-1 space-y-1">
          <VersionRowList
            items={items}
            changeRequestApprovalConfig={changeRequestApprovalConfig}
            onUseVersion={onUseVersion}
            onViewDiff={onViewDiff}
          />
        </div>
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
  displayedLiveVersions,
  liveCanvasVersionId,
  liveVersions,
  loadMoreLiveVersionsDisabled,
  liveVersionChangeRequestsByVersionId,
  onLoadMoreLiveVersions,
  selectedVersionId,
}: {
  displayedLiveVersions: CanvasesCanvasVersion[];
  liveCanvasVersionId?: string;
  liveVersions: CanvasesCanvasVersion[];
  loadMoreLiveVersionsDisabled?: boolean;
  liveVersionChangeRequestsByVersionId?: Map<string, CanvasesCanvasChangeRequest>;
  onLoadMoreLiveVersions?: () => void;
  selectedVersionId: string;
}): VersionRowItem[] {
  return displayedLiveVersions.map((version, index) => {
    const versionID = version.metadata?.id || "";
    const isFirstCanvasVersion =
      index === liveVersions.length - 1 && (onLoadMoreLiveVersions ? !!loadMoreLiveVersionsDisabled : true);
    return {
      key: versionID,
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

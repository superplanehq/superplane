import type { CanvasChangeManagement, CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ChevronDown, ChevronRight, History } from "lucide-react";
import type { KeyboardEvent as ReactKeyboardEvent, MouseEvent as ReactMouseEvent } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { getChangeRequestReviewPhase } from "./changeRequestReviewActions";
import { formatVersionLabel, formatVersionTimestamp, isDraftVersion, isPublishedVersion } from "./lib/canvas-versions";

const CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY = "canvasVersionControlSidebarWidth";
const DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 460;
const LEGACY_DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 340;
const MIN_CANVAS_VERSION_CONTROL_WIDTH = 280;
const MAX_CANVAS_VERSION_CONTROL_WIDTH = 640;
const INITIAL_VISIBLE_LIVE_VERSIONS = 5;
const LOAD_OLDER_LIVE_VERSIONS_STEP = 5;

interface CanvasVersionControlSidebarProps {
  isOpen: boolean;
  selectedVersionId: string;
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
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
  changeRequestApprovalConfig?: CanvasChangeManagement;
}

export function CanvasVersionControlSidebar({
  isOpen,
  selectedVersionId,
  pendingApprovalVersions,
  rejectedVersions,
  liveVersions,
  liveVersionChangeRequestsByVersionId,
  canUpdateCanvas,
  isTemplate,
  canvasDeletedRemotely,
  onUseVersion,
  onLoadMoreLiveVersions,
  loadMoreLiveVersionsDisabled,
  loadMoreLiveVersionsPending,
  changeRequestApprovalConfig,
}: CanvasVersionControlSidebarProps) {
  const rejectedList = rejectedVersions ?? [];

  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") {
      return DEFAULT_CANVAS_VERSION_CONTROL_WIDTH;
    }

    const stored = window.localStorage.getItem(CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY);
    const parsed = stored ? Number.parseInt(stored, 10) : NaN;
    if (!Number.isFinite(parsed)) {
      return DEFAULT_CANVAS_VERSION_CONTROL_WIDTH;
    }

    if (parsed === LEGACY_DEFAULT_CANVAS_VERSION_CONTROL_WIDTH) {
      return DEFAULT_CANVAS_VERSION_CONTROL_WIDTH;
    }

    return Math.max(MIN_CANVAS_VERSION_CONTROL_WIDTH, Math.min(MAX_CANVAS_VERSION_CONTROL_WIDTH, parsed));
  });
  const [isResizing, setIsResizing] = useState(false);
  const [isResizeHandleHovered, setIsResizeHandleHovered] = useState(false);
  const [rejectedVersionsExpanded, setRejectedVersionsExpanded] = useState(false);
  const [visibleLiveVersionCount, setVisibleLiveVersionCount] = useState(INITIAL_VISIBLE_LIVE_VERSIONS);
  const sidebarRef = useRef<HTMLElement>(null);

  const hasSelectedVersion = !!selectedVersionId;
  useEffect(() => {
    setVisibleLiveVersionCount(INITIAL_VISIBLE_LIVE_VERSIONS);
  }, [liveVersions[0]?.metadata?.id]);

  const displayedLiveVersions = liveVersions.slice(0, visibleLiveVersionCount);
  const hasVisiblePendingSelection = hasSelectedVersion
    ? (pendingApprovalVersions || []).some((item) => item.version.metadata?.id === selectedVersionId)
    : false;
  const hasVisibleLiveSelection = hasSelectedVersion
    ? displayedLiveVersions.some((version) => version.metadata?.id === selectedVersionId)
    : false;
  const hasVisibleSelection = hasVisiblePendingSelection || hasVisibleLiveSelection;
  const canExpandLocal = visibleLiveVersionCount < liveVersions.length;
  const showLoadOlderVersions = canExpandLocal || !!onLoadMoreLiveVersions;

  const handleLoadOlderVersions = useCallback(() => {
    if (canExpandLocal) {
      setVisibleLiveVersionCount((prev) => Math.min(prev + LOAD_OLDER_LIVE_VERSIONS_STEP, liveVersions.length));
      return;
    }
    onLoadMoreLiveVersions?.();
  }, [canExpandLocal, liveVersions.length, onLoadMoreLiveVersions]);

  const loadOlderVersionsDisabled = !canExpandLocal && (loadMoreLiveVersionsDisabled ?? !onLoadMoreLiveVersions);
  const loadOlderVersionsPending = !canExpandLocal && !!loadMoreLiveVersionsPending;

  const handleMouseDown = useCallback((event: ReactMouseEvent<HTMLDivElement>) => {
    event.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMove = useCallback(
    (event: MouseEvent) => {
      if (!isResizing) {
        return;
      }

      const sidebarLeft = sidebarRef.current?.getBoundingClientRect().left ?? 0;
      const newWidth = event.clientX - sidebarLeft;
      const clampedWidth = Math.max(
        MIN_CANVAS_VERSION_CONTROL_WIDTH,
        Math.min(MAX_CANVAS_VERSION_CONTROL_WIDTH, newWidth),
      );
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    window.localStorage.setItem(CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    if (!isResizing) {
      return;
    }

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.cursor = "ew-resize";
    document.body.style.userSelect = "none";

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing, handleMouseMove, handleMouseUp]);

  if (!isOpen) {
    return null;
  }

  return (
    <aside
      ref={sidebarRef}
      className={cn(
        "z-20 h-full border-r bg-white relative",
        isResizeHandleHovered || isResizing ? "border-slate-800" : "border-slate-950/15",
      )}
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
      <div
        onMouseDown={handleMouseDown}
        onMouseEnter={() => setIsResizeHandleHovered(true)}
        onMouseLeave={() => setIsResizeHandleHovered(false)}
        className="absolute right-0 top-0 bottom-0 w-4 cursor-ew-resize bg-transparent transition-colors z-30"
        style={{ marginRight: "-8px" }}
      />
      <div className="flex h-full flex-col">
        <div className="flex h-12 items-center px-4">
          <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
            <History className="h-4 w-4" />
            Version History
          </div>
        </div>

        <div className="flex-1 overflow-auto p-3">
          {!canUpdateCanvas && !canvasDeletedRemotely ? (
            <p className="text-xs text-slate-600">You do not have permission to edit this canvas.</p>
          ) : null}
          {canvasDeletedRemotely ? (
            <p className="text-xs text-red-700">This canvas was deleted from another session.</p>
          ) : null}
          {isTemplate ? <p className="text-xs text-slate-600">Template canvases are read-only.</p> : null}

          <section className="mt-3 rounded-md">
            {liveVersions.length === 0 &&
            (pendingApprovalVersions?.length || 0) === 0 &&
            (rejectedVersions?.length || 0) === 0 ? (
              <p className="mt-2 text-xs text-slate-600">No published history yet.</p>
            ) : (
              <>
                <div className="-mt-4 space-y-1">
                  {(pendingApprovalVersions || []).map((item) => {
                    const versionID = item.version.metadata?.id || "";
                    const isActive = versionID === selectedVersionId;

                    return (
                      <VersionRow
                        key={`pending-${versionID || item.changeRequest.metadata?.id || "unknown"}`}
                        rowTestId="canvas-pending-change-request-version-row"
                        version={item.version}
                        changeRequest={item.changeRequest}
                        changeRequestApprovalConfig={changeRequestApprovalConfig}
                        isActive={isActive}
                        isLatestPublished={false}
                        onUseVersion={onUseVersion}
                      />
                    );
                  })}
                  {displayedLiveVersions.map((version, index) => {
                    const versionID = version.metadata?.id || "";
                    const isLatestPublished = index === 0;
                    const isActive = versionID === selectedVersionId || (isLatestPublished && !hasVisibleSelection);
                    const changeRequest = versionID ? liveVersionChangeRequestsByVersionId?.get(versionID) : undefined;
                    return (
                      <VersionRow
                        key={versionID}
                        version={version}
                        changeRequest={changeRequest}
                        changeRequestApprovalConfig={changeRequestApprovalConfig}
                        isActive={isActive}
                        isLatestPublished={isLatestPublished}
                        onUseVersion={onUseVersion}
                      />
                    );
                  })}
                </div>
                {showLoadOlderVersions ? (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2 w-fit self-start"
                    onClick={handleLoadOlderVersions}
                    disabled={loadOlderVersionsDisabled}
                  >
                    {loadOlderVersionsPending ? "Loading..." : "Load older versions"}
                  </Button>
                ) : null}
              </>
            )}
            {rejectedList.length > 0 ? (
              <div className="mt-3 border-t border-slate-200 pt-3">
                <button
                  type="button"
                  className="flex w-full items-center gap-1 rounded-md py-1.5 text-left text-xs font-medium text-slate-500"
                  onClick={() => setRejectedVersionsExpanded((v) => !v)}
                  aria-expanded={rejectedVersionsExpanded}
                >
                  {rejectedVersionsExpanded ? (
                    <ChevronDown className="h-4 w-4 shrink-0" aria-hidden />
                  ) : (
                    <ChevronRight className="h-4 w-4 shrink-0" aria-hidden />
                  )}
                  <span>Rejected ({rejectedList.length})</span>
                </button>
                {rejectedVersionsExpanded ? (
                  <div className="mt-1 space-y-1">
                    {rejectedList.map((item) => {
                      const versionID = item.version.metadata?.id || "";
                      const isActive = versionID === selectedVersionId;

                      return (
                        <VersionRow
                          key={`rejected-${versionID || item.changeRequest.metadata?.id || "unknown"}`}
                          version={item.version}
                          changeRequest={item.changeRequest}
                          changeRequestApprovalConfig={changeRequestApprovalConfig}
                          variant="rejected"
                          isActive={isActive}
                          isLatestPublished={false}
                          onUseVersion={onUseVersion}
                        />
                      );
                    })}
                  </div>
                ) : null}
              </div>
            ) : null}
          </section>
        </div>
      </div>
    </aside>
  );
}

function VersionRow({
  version,
  changeRequest,
  changeRequestApprovalConfig,
  variant = "default",
  isActive = false,
  isLatestPublished = false,
  rowTestId,
  onUseVersion,
}: {
  version: CanvasesCanvasVersion;
  changeRequest?: CanvasesCanvasChangeRequest;
  changeRequestApprovalConfig?: CanvasChangeManagement;
  variant?: "default" | "rejected";
  isActive?: boolean;
  isLatestPublished?: boolean;
  /** Stable hook for E2E: pending approval rows only. */
  rowTestId?: string;
  onUseVersion: (versionID: string) => void;
}) {
  const versionID = version.metadata?.id;
  const ownerName = version.metadata?.owner?.name || "Unknown owner";
  const changeRequestTitle = changeRequest?.metadata?.title?.trim();
  const versionLabel = isDraftVersion(version) ? "Draft" : changeRequestTitle || formatVersionLabel(version);
  const versionTimestamp = formatVersionTimestamp(version);
  const versionSubtitle =
    isPublishedVersion(version) && !changeRequestTitle
      ? ownerName
      : versionTimestamp
        ? `${ownerName} on ${versionTimestamp}`
        : ownerName;
  const reviewPhase = getChangeRequestReviewPhase(changeRequest, changeRequestApprovalConfig);
  const activeReviewPhase = variant === "rejected" ? null : reviewPhase.kind === "none" ? null : reviewPhase;

  if (!versionID) {
    return null;
  }

  const handleRowActivate = () => {
    onUseVersion(versionID);
  };

  const handleRowKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onUseVersion(versionID);
    }
  };

  return (
    <div
      data-testid={rowTestId}
      className={cn(
        "w-full rounded-md px-2.5 py-2 text-left transition cursor-pointer",
        isActive
          ? variant === "rejected"
            ? "bg-red-50"
            : activeReviewPhase
              ? activeReviewPhase.sidebarRowActiveClassName
              : "bg-sky-100"
          : "bg-white hover:bg-slate-100",
      )}
      role="button"
      tabIndex={0}
      onClick={handleRowActivate}
      onKeyDown={handleRowKeyDown}
      aria-label={`Preview ${versionLabel}`}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-slate-900">{versionLabel}</p>
          <p className="mt-0.5 text-xs text-slate-600 truncate">
            {isLatestPublished ? (
              <>
                <span className="font-medium text-sky-600">Current Version</span> {"\u00b7"}{" "}
              </>
            ) : null}
            {variant === "rejected" ? (
              <>
                <span className="font-medium text-red-600">Rejected</span> {"\u00b7"}{" "}
              </>
            ) : activeReviewPhase ? (
              <>
                <span className={cn("mr-1", activeReviewPhase.dotClassName)}>{"\u25cf"}</span>
                <span className={activeReviewPhase.labelClassName}>{activeReviewPhase.label}</span> {"\u00b7"}{" "}
              </>
            ) : null}
            {versionSubtitle}
          </p>
        </div>
      </div>
    </div>
  );
}

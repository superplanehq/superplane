import { CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { ChevronLeft, GitBranch, GitPullRequest, RotateCcw } from "lucide-react";
import { MouseEvent as ReactMouseEvent, ReactNode, useCallback, useEffect, useRef, useState } from "react";

const CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY = "canvasVersionControlSidebarWidth";
const DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 460;
const LEGACY_DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 340;
const MIN_CANVAS_VERSION_CONTROL_WIDTH = 280;
const MAX_CANVAS_VERSION_CONTROL_WIDTH = 640;

interface CanvasVersionControlSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  liveCanvasVersionId?: string;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  liveVersions: CanvasesCanvasVersion[];
  liveVersionsTotalCount?: number;
  canUpdateCanvas: boolean;
  isTemplate: boolean;
  hasEditableVersion: boolean;
  isEditModeEnabled: boolean;
  canvasDeletedRemotely: boolean;
  onToggleEditMode: () => void;
  onResetDraft: () => void;
  onUseVersion: (versionID: string) => void;
  onCreateChangeRequest: () => void;
  onLoadMoreLiveVersions?: () => void;
  toggleEditModeDisabled: boolean;
  toggleEditModeDisabledTooltip?: string;
  resetDraftDisabled: boolean;
  resetDraftDisabledTooltip?: string;
  createChangeRequestDisabled: boolean;
  createChangeRequestDisabledTooltip?: string;
  loadMoreLiveVersionsDisabled?: boolean;
  toggleEditModePending: boolean;
  resetDraftPending: boolean;
  createChangeRequestPending: boolean;
  loadMoreLiveVersionsPending?: boolean;
}

function formatVersionTimestamp(version?: CanvasesCanvasVersion): string | undefined {
  const raw = version?.metadata?.updatedAt || version?.metadata?.publishedAt || version?.metadata?.createdAt;
  if (!raw) {
    return undefined;
  }

  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }

  return date.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function formatVersionLabel(version?: CanvasesCanvasVersion): string {
  if (version?.metadata?.isPublished) {
    return "Published version";
  }

  return "Draft version";
}

function formatVersionLabelWithTimestamp(version?: CanvasesCanvasVersion): string {
  const label = formatVersionLabel(version);
  const timestamp = formatVersionTimestamp(version);
  if (!timestamp) {
    return label;
  }

  return `${label} · ${timestamp}`;
}

function withTooltip(disabled: boolean, message: string | undefined, element: ReactNode): ReactNode {
  if (!disabled || !message) {
    return element;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="w-full">{element}</div>
      </TooltipTrigger>
      <TooltipContent side="right">{message}</TooltipContent>
    </Tooltip>
  );
}

export function CanvasVersionControlSidebar({
  isOpen,
  onToggle,
  liveCanvasVersionId,
  selectedCanvasVersion,
  liveVersions,
  liveVersionsTotalCount,
  canUpdateCanvas,
  isTemplate,
  hasEditableVersion,
  isEditModeEnabled,
  canvasDeletedRemotely,
  onToggleEditMode,
  onResetDraft,
  onUseVersion,
  onCreateChangeRequest,
  onLoadMoreLiveVersions,
  toggleEditModeDisabled,
  toggleEditModeDisabledTooltip,
  resetDraftDisabled,
  resetDraftDisabledTooltip,
  createChangeRequestDisabled,
  createChangeRequestDisabledTooltip,
  loadMoreLiveVersionsDisabled,
  toggleEditModePending,
  resetDraftPending,
  createChangeRequestPending,
  loadMoreLiveVersionsPending,
}: CanvasVersionControlSidebarProps) {
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "";

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
  const sidebarRef = useRef<HTMLElement>(null);

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
      className="z-20 h-full border-r border-slate-950/10 bg-white relative"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
    >
      <div
        onMouseDown={handleMouseDown}
        className={`absolute right-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-slate-100 transition-colors flex items-center justify-center group z-30 ${
          isResizing ? "bg-sky-50" : ""
        }`}
        style={{ marginRight: "-8px" }}
      >
        <div
          className={`h-14 w-2 rounded-full bg-slate-300 transition-colors ${
            isResizing ? "bg-sky-500" : "group-hover:bg-slate-600"
          }`}
        />
      </div>
      <div className="flex h-full flex-col">
        <div className="flex h-12 items-center justify-between border-b border-slate-200 px-3">
          <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
            <GitBranch className="h-4 w-4" />
            Version Control
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            className="h-7 w-7"
            onClick={() => onToggle(false)}
            aria-label="Collapse version control"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
        </div>

        <div className="flex-1 overflow-auto p-3">
          <section className="rounded-md border border-slate-200 p-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">Actions</p>
            <div className="mt-2 flex flex-col gap-2">
              {withTooltip(
                toggleEditModeDisabled,
                toggleEditModeDisabledTooltip,
                <div className="flex items-center justify-between gap-3 rounded-md border border-slate-200 px-3 py-2">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-slate-900">Edit mode</p>
                    <p className="text-xs text-slate-600">
                      {toggleEditModePending
                        ? "Updating..."
                        : isEditModeEnabled
                          ? "Enabled"
                          : "Enable to create or use your draft"}
                    </p>
                  </div>
                  <Switch
                    checked={isEditModeEnabled}
                    onCheckedChange={() => onToggleEditMode()}
                    disabled={toggleEditModeDisabled || toggleEditModePending}
                  />
                </div>,
              )}

              {isEditModeEnabled
                ? withTooltip(
                    resetDraftDisabled,
                    resetDraftDisabledTooltip,
                    <Button
                      onClick={onResetDraft}
                      disabled={resetDraftDisabled}
                      className="w-full justify-start min-w-0"
                      variant="outline"
                    >
                      <RotateCcw className="h-4 w-4" />
                      <span className="truncate min-w-0">
                        {resetDraftPending ? "Resetting draft..." : "Reset Draft Changes"}
                      </span>
                    </Button>,
                  )
                : null}

              {hasEditableVersion &&
                withTooltip(
                  createChangeRequestDisabled,
                  createChangeRequestDisabledTooltip,
                  <Button
                    onClick={onCreateChangeRequest}
                    disabled={createChangeRequestDisabled}
                    className="w-full justify-start min-w-0"
                    variant="outline"
                  >
                    <GitPullRequest className="h-4 w-4" />
                    <span className="truncate min-w-0">
                      {createChangeRequestPending ? "Publishing change request..." : "Create change request"}
                    </span>
                  </Button>,
                )}

              {!canUpdateCanvas && !canvasDeletedRemotely ? (
                <p className="text-xs text-slate-600">You do not have permission to edit this canvas.</p>
              ) : null}
              {canvasDeletedRemotely ? (
                <p className="text-xs text-red-700">This canvas was deleted from another session.</p>
              ) : null}
              {isTemplate ? <p className="text-xs text-slate-600">Template canvases are read-only.</p> : null}
            </div>
          </section>

          <section className="mt-3 rounded-md border border-slate-200 p-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">
              Live History ({liveVersionsTotalCount ?? liveVersions.length})
            </p>
            {liveVersions.length === 0 ? (
              <p className="mt-2 text-xs text-slate-600">No published history yet.</p>
            ) : (
              <>
                <div className="mt-2 space-y-2">
                  {liveVersions.map((version) => {
                    const versionID = version.metadata?.id || "";
                    const isActive = versionID === selectedVersionId;
                    const isCurrentLive = liveCanvasVersionId === versionID;
                    return (
                      <VersionRow
                        key={versionID}
                        version={version}
                        isActive={isActive}
                        subtitle={isCurrentLive ? "Current live" : "Live history"}
                        onUseVersion={onUseVersion}
                      />
                    );
                  })}
                </div>
                {onLoadMoreLiveVersions ? (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2 w-full"
                    onClick={onLoadMoreLiveVersions}
                    disabled={loadMoreLiveVersionsDisabled}
                  >
                    {loadMoreLiveVersionsPending ? "Loading..." : "Load older versions"}
                  </Button>
                ) : null}
              </>
            )}
          </section>
        </div>
      </div>
    </aside>
  );
}

function VersionRow({
  version,
  isActive = false,
  subtitle,
  onUseVersion,
}: {
  version: CanvasesCanvasVersion;
  isActive?: boolean;
  subtitle?: string;
  onUseVersion: (versionID: string) => void;
}) {
  const versionID = version.metadata?.id;
  const ownerName = version.metadata?.owner?.name || "Unknown owner";
  const versionLabel = formatVersionLabelWithTimestamp(version);

  if (!versionID) {
    return null;
  }

  return (
    <button
      type="button"
      onClick={() => onUseVersion(versionID)}
      className={cn(
        "w-full rounded-md border px-2.5 py-2 text-left transition",
        isActive ? "border-sky-300 bg-sky-50" : "border-slate-200 bg-white",
        "hover:border-slate-300 hover:bg-slate-50",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium text-slate-900 truncate flex-1 min-w-0">{versionLabel}</p>
        <div className="flex items-center gap-1.5">
          {isActive ? (
            <span className="rounded bg-sky-600 px-1.5 py-0.5 text-[10px] font-medium text-white max-w-[72px] truncate whitespace-nowrap">
              Active
            </span>
          ) : null}
        </div>
      </div>
      <p className="mt-0.5 text-xs text-slate-600 truncate">{subtitle || ownerName}</p>
    </button>
  );
}

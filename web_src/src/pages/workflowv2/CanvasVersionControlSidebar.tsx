import { CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { ChevronLeft, CircleDot, GitBranch, Plus, Rocket, Trash2, User } from "lucide-react";
import { MouseEvent as ReactMouseEvent, ReactNode, useCallback, useEffect, useRef, useState } from "react";

const CANVAS_VERSION_CONTROL_WIDTH_STORAGE_KEY = "canvasVersionControlSidebarWidth";
const DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 460;
const LEGACY_DEFAULT_CANVAS_VERSION_CONTROL_WIDTH = 340;
const MIN_CANVAS_VERSION_CONTROL_WIDTH = 280;
const MAX_CANVAS_VERSION_CONTROL_WIDTH = 640;

interface CanvasVersionControlSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  currentUserId?: string;
  activeCanvasVersionId?: string;
  liveCanvasVersionId?: string;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  liveCanvasVersion?: CanvasesCanvasVersion;
  draftVersions: CanvasesCanvasVersion[];
  canUpdateCanvas: boolean;
  isTemplate: boolean;
  hasEditableVersion: boolean;
  hasUnsavedChanges: boolean;
  canvasDeletedRemotely: boolean;
  onCreateVersion: () => void;
  onPublishVersion: () => void;
  onDiscardVersion: () => void;
  onUseVersion: (versionID: string) => void;
  createVersionDisabled: boolean;
  createVersionDisabledTooltip?: string;
  publishVersionDisabled: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled: boolean;
  discardVersionDisabledTooltip?: string;
  createVersionPending: boolean;
  publishVersionPending: boolean;
  discardVersionPending: boolean;
}

function isSameUserID(left?: string, right?: string): boolean {
  if (!left || !right) {
    return false;
  }

  return left.trim().toLowerCase() === right.trim().toLowerCase();
}

function getVersionRevision(version?: CanvasesCanvasVersion): number {
  return version?.metadata?.revision ?? 0;
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
  const revision = version?.metadata?.revision ?? "?";
  return `Revision ${revision}`;
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
  currentUserId,
  activeCanvasVersionId,
  liveCanvasVersionId,
  selectedCanvasVersion,
  liveCanvasVersion,
  draftVersions,
  canUpdateCanvas,
  isTemplate,
  hasEditableVersion,
  hasUnsavedChanges,
  canvasDeletedRemotely,
  onCreateVersion,
  onPublishVersion,
  onDiscardVersion,
  onUseVersion,
  createVersionDisabled,
  createVersionDisabledTooltip,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  createVersionPending,
  publishVersionPending,
  discardVersionPending,
}: CanvasVersionControlSidebarProps) {
  const selectedVersionId = selectedCanvasVersion?.metadata?.id || liveCanvasVersionId || "";
  const selectedRevisionLabel = formatVersionLabelWithTimestamp(selectedCanvasVersion || liveCanvasVersion);
  const selectedOwner = selectedCanvasVersion?.metadata?.owner?.name;
  const currentUserDrafts = draftVersions.filter((version) => isSameUserID(version.metadata?.owner?.id, currentUserId));
  const currentUserLatestDraft = [...currentUserDrafts].sort(
    (a, b) => getVersionRevision(b) - getVersionRevision(a),
  )[0];
  const currentUserLatestDraftID = currentUserLatestDraft?.metadata?.id;
  const canSwitchToLive = !!liveCanvasVersionId && selectedVersionId !== liveCanvasVersionId;
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
          <section className="rounded-md border border-slate-200 bg-slate-50/70 p-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">Current</p>
            <div className="mt-2 flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-sm font-semibold text-slate-900 break-words">
                  {hasEditableVersion ? "Draft version" : "Live version"} {selectedRevisionLabel}
                </p>
                <p className="mt-0.5 text-xs text-slate-600 break-words">
                  {selectedOwner ? `Owner: ${selectedOwner}` : "Published canvas"}
                </p>
              </div>
              <span
                className={cn(
                  "rounded-full px-2 py-0.5 text-[11px] font-medium max-w-[140px] truncate whitespace-nowrap",
                  hasEditableVersion ? "bg-amber-100 text-amber-800" : "bg-emerald-100 text-emerald-800",
                )}
              >
                {hasEditableVersion ? "Editing" : "Read-only"}
              </span>
            </div>
            {hasEditableVersion && hasUnsavedChanges ? (
              <p className="mt-2 text-xs text-amber-700">You have unsaved draft changes.</p>
            ) : null}
            {canvasDeletedRemotely ? (
              <p className="mt-2 text-xs text-red-700">This canvas was deleted from another session.</p>
            ) : null}
            {isTemplate ? <p className="mt-2 text-xs text-slate-600">Template canvases are read-only.</p> : null}
          </section>

          <section className="mt-3 rounded-md border border-slate-200 p-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">Actions</p>
            <div className="mt-2 flex flex-col gap-2">
              {withTooltip(
                createVersionDisabled,
                createVersionDisabledTooltip,
                <Button
                  onClick={onCreateVersion}
                  disabled={createVersionDisabled}
                  className="w-full justify-start min-w-0"
                >
                  <Plus className="h-4 w-4" />
                  <span className="truncate min-w-0">
                    {createVersionPending ? "Creating version..." : "Create version from live"}
                  </span>
                </Button>,
              )}

              {hasEditableVersion &&
                withTooltip(
                  publishVersionDisabled,
                  publishVersionDisabledTooltip,
                  <Button
                    onClick={onPublishVersion}
                    disabled={publishVersionDisabled}
                    className="w-full justify-start min-w-0"
                    variant="default"
                  >
                    <Rocket className="h-4 w-4" />
                    <span className="truncate min-w-0">
                      {publishVersionPending ? "Publishing..." : "Publish current version"}
                    </span>
                  </Button>,
                )}

              {hasEditableVersion &&
                withTooltip(
                  discardVersionDisabled,
                  discardVersionDisabledTooltip,
                  <Button
                    onClick={onDiscardVersion}
                    disabled={discardVersionDisabled}
                    className="w-full justify-start text-red-700 border-red-200 hover:text-red-800 hover:border-red-300 min-w-0"
                    variant="outline"
                  >
                    <Trash2 className="h-4 w-4" />
                    <span className="truncate min-w-0">
                      {discardVersionPending ? "Discarding..." : "Discard current draft"}
                    </span>
                  </Button>,
                )}

              {canSwitchToLive && liveCanvasVersionId ? (
                <Button
                  variant="outline"
                  className="w-full justify-start min-w-0"
                  onClick={() => onUseVersion(liveCanvasVersionId)}
                >
                  <CircleDot className="h-4 w-4 text-emerald-600" />
                  <span className="truncate min-w-0">
                    Switch to live {formatVersionLabelWithTimestamp(liveCanvasVersion)}
                  </span>
                </Button>
              ) : null}

              {!hasEditableVersion && currentUserLatestDraftID ? (
                <Button
                  variant="outline"
                  className="w-full justify-start min-w-0"
                  onClick={() => onUseVersion(currentUserLatestDraftID)}
                >
                  <User className="h-4 w-4" />
                  <span className="truncate min-w-0">
                    Switch to my draft {formatVersionLabelWithTimestamp(currentUserLatestDraft)}
                  </span>
                </Button>
              ) : null}

              {!canUpdateCanvas && !canvasDeletedRemotely ? (
                <p className="text-xs text-slate-600">You do not have permission to edit this canvas.</p>
              ) : null}
            </div>
          </section>

          <section className="mt-3 rounded-md border border-slate-200 p-3">
            <div className="flex items-center justify-between">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">Live</p>
              {liveCanvasVersion ? (
                <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-800">
                  Active
                </span>
              ) : null}
            </div>
            {liveCanvasVersion ? (
              <VersionRow
                version={liveCanvasVersion}
                isActive={selectedVersionId === liveCanvasVersion.metadata?.id}
                subtitle="Published"
                onUseVersion={onUseVersion}
              />
            ) : (
              <p className="mt-2 text-xs text-slate-600">No live version available.</p>
            )}
          </section>

          <section className="mt-3 rounded-md border border-slate-200 p-3">
            <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">
              Drafts ({draftVersions.length})
            </p>
            {draftVersions.length === 0 ? (
              <p className="mt-2 text-xs text-slate-600">No draft versions yet.</p>
            ) : (
              <div className="mt-2 space-y-2">
                {draftVersions.map((version) => {
                  const versionID = version.metadata?.id || "";
                  const isActive = versionID === activeCanvasVersionId;
                  return (
                    <VersionRow
                      key={versionID}
                      version={version}
                      isActive={isActive}
                      subtitle="Your draft"
                      onUseVersion={onUseVersion}
                    />
                  );
                })}
              </div>
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
  const revisionLabel = formatVersionLabelWithTimestamp(version);

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
        <p className="text-sm font-medium text-slate-900 truncate flex-1 min-w-0">{revisionLabel}</p>
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

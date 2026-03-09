import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import {
  CloudAlert,
  CloudCheck,
  CloudUpload,
  Home,
  ChevronDown,
  LogOut,
  Palette,
  RotateCcw,
  Undo2,
  SquarePen,
  Pencil,
  Rocket,
} from "lucide-react";
import { Button } from "../button";
import { Switch } from "../switch";
import { useCanvases } from "@/hooks/useCanvasData";
import { Link, useParams } from "react-router-dom";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
  href?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
}

type HeaderMode = "default" | "version-live" | "version-edit" | "versioning-disabled";
type SaveState = "saved" | "saving" | "unsaved";

interface HeaderProps {
  breadcrumbs: BreadcrumbItem[];
  onSave?: () => void;
  onCreateVersion?: () => void;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  onUndo?: () => void;
  canUndo?: boolean;
  onLogoClick?: () => void;
  organizationId?: string;
  versionLabel?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  createVersionDisabled?: boolean;
  createVersionDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  isAutoSaveEnabled?: boolean;
  onToggleAutoSave?: () => void;
  autoSaveDisabled?: boolean;
  autoSaveDisabledTooltip?: string;
  onExportYamlCopy?: () => void;
  onExportYamlDownload?: () => void;
  topViewMode?: "canvas" | "yaml" | "memory" | "settings" | "versioning";
  onTopViewModeChange?: (mode: "canvas" | "yaml" | "memory" | "settings" | "versioning") => void;
  showVersioningTab?: boolean;
  memoryItemCount?: number;
  versioningItemCount?: number;
  mode?: HeaderMode;
  saveState?: SaveState;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  versioningDisabledTooltip?: string;
  showPendingDraftBadge?: boolean;
}

export function Header({
  breadcrumbs,
  onSave,
  onPublishVersion,
  onDiscardVersion,
  onUndo,
  canUndo,
  onLogoClick,
  organizationId,
  unsavedMessage,
  saveIsPrimary,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  isAutoSaveEnabled,
  onToggleAutoSave,
  autoSaveDisabled,
  autoSaveDisabledTooltip,
  onExportYamlCopy,
  onExportYamlDownload,
  topViewMode,
  onTopViewModeChange,
  showVersioningTab = true,
  memoryItemCount,
  versioningItemCount,
  mode = "default",
  saveState = "saved",
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  versioningDisabledTooltip,
  showPendingDraftBadge,
}: HeaderProps) {
  const { workflowId } = useParams<{ workflowId?: string }>();
  const { data: workflows = [], isLoading: workflowsLoading } = useCanvases(organizationId || "");
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isYamlMenuOpen, setIsYamlMenuOpen] = useState(false);
  const [isEditingMenuOpen, setIsEditingMenuOpen] = useState(false);
  const [isSaveMenuOpen, setIsSaveMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement | null>(null);

  // Get the workflow name from the workflows list if workflowId is available
  // Otherwise, use breadcrumbs[1] which is always the workflow name (index 0 is "Canvases")
  // Fall back to the last breadcrumb item if neither is available
  const currentWorkflowName = (() => {
    if (workflowId) {
      const workflow = workflows.find((w) => w.metadata?.id === workflowId);
      if (workflow?.metadata?.name) {
        return workflow.metadata.name;
      }
    }
    // breadcrumbs[1] is always the workflow name (index 0 is "Canvases")
    if (breadcrumbs.length > 1 && breadcrumbs[1]?.label) {
      return breadcrumbs[1].label;
    }
    // Fall back to last breadcrumb if no workflow name found
    return breadcrumbs.length > 0 ? breadcrumbs[breadcrumbs.length - 1].label : "";
  })();

  useEffect(() => {
    if (!isMenuOpen) return;

    const handlePointerDown = (event: MouseEvent | TouchEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsMenuOpen(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMenuOpen(false);
      }
    };

    const listenerOptions: AddEventListenerOptions = { capture: true };

    document.addEventListener("mousedown", handlePointerDown, listenerOptions);
    document.addEventListener("touchstart", handlePointerDown, listenerOptions);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handlePointerDown, listenerOptions);
      document.removeEventListener("touchstart", handlePointerDown, listenerOptions);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isMenuOpen]);

  const wrapWithTooltip = (disabled: boolean | undefined, message: string | undefined, child: ReactNode) => {
    if (!disabled || !message) return child;
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{child}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{message}</TooltipContent>
      </Tooltip>
    );
  };

  const isDefaultMode = mode === "default";
  const showEditButton = mode === "version-live";
  const showEditingDropdown = mode === "version-edit";
  const showVersioningDisabledBadge = mode === "versioning-disabled";
  const showSaveDropdown = mode === "version-edit" || mode === "versioning-disabled";
  const showSaveUndoActions = showSaveDropdown && !isAutoSaveEnabled && saveState === "unsaved";
  const autoSaveToggleDisabled = autoSaveDisabled || !onToggleAutoSave;
  const saveStatusLabel = saveState === "saving" ? "Saving..." : saveState === "unsaved" ? "Unsaved" : "Saved";
  const saveStatusIcon =
    saveState === "saving" ? (
      <CloudUpload className="h-4 w-4 animate-pulse text-sky-600" />
    ) : saveState === "unsaved" ? (
      <CloudAlert className="h-4 w-4 text-amber-600" />
    ) : (
      <CloudCheck className="h-4 w-4 text-emerald-600" />
    );

  return (
    <>
      <header className="bg-white border-b border-slate-950/15">
        <div className="relative grid h-12 grid-cols-3 items-center px-4">
          <div className="flex items-center gap-3 justify-self-start">
            <OrganizationMenuButton organizationId={organizationId} onLogoClick={onLogoClick} />

            {/* Canvas Dropdown */}
            {organizationId && (
              <div className="relative flex items-center" ref={menuRef}>
                <button
                  onClick={() => setIsMenuOpen((prev) => !prev)}
                  className="flex items-center gap-1 cursor-pointer h-7"
                  aria-label="Open canvas menu"
                  aria-expanded={isMenuOpen}
                  disabled={workflowsLoading}
                >
                  <span className="text-sm text-gray-800 font-medium">
                    {currentWorkflowName || (workflowsLoading ? "Loading..." : "Select canvas")}
                  </span>
                  <ChevronDown
                    size={16}
                    className={`text-gray-400 transition-transform ${isMenuOpen ? "rotate-180" : ""}`}
                  />
                </button>
                {isMenuOpen && !workflowsLoading && (
                  <div className="absolute left-0 top-13 z-50 min-w-[15rem] w-max rounded-md outline outline-slate-950/20 bg-white shadow-lg">
                    <div className="px-4 pt-3 pb-4">
                      {/* All Canvases Link */}
                      <div className="mb-2">
                        <Link
                          to={organizationId ? `/${organizationId}` : "/"}
                          className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-gray-500 hover:text-gray-800"
                          onClick={() => setIsMenuOpen(false)}
                        >
                          <Home size={16} className="text-gray-500 transition group-hover:text-gray-800" />
                          <span>All Canvases</span>
                        </Link>
                      </div>
                      {/* Divider */}
                      <div className="border-b border-gray-300 mb-2"></div>
                      {/* Canvas List */}
                      <div className="mt-2 flex flex-col">
                        {workflows.map((workflow) => {
                          const isSelected = workflow.metadata?.id === workflowId;
                          const workflowHref =
                            workflow.metadata?.id && organizationId
                              ? `/${organizationId}/canvases/${workflow.metadata.id}`
                              : undefined;
                          if (!workflowHref) {
                            return (
                              <span
                                key={workflow.metadata?.id}
                                className="group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-left text-gray-500"
                              >
                                <span className="w-4" />
                                <span>{workflow.metadata?.name || "Unnamed Canvas"}</span>
                              </span>
                            );
                          }

                          return (
                            <Link
                              key={workflow.metadata?.id}
                              to={workflowHref}
                              onClick={() => setIsMenuOpen(false)}
                              className={`group flex items-center gap-2 rounded-md px-1.5 py-1 text-sm font-medium text-left ${
                                isSelected
                                  ? "bg-sky-100 text-gray-800"
                                  : "text-gray-500 hover:bg-sky-100 hover:text-gray-800"
                              }`}
                            >
                              {isSelected ? (
                                <Palette size={16} className="text-gray-800 transition group-hover:text-gray-800" />
                              ) : (
                                <span className="w-4" />
                              )}
                              <span>{workflow.metadata?.name || "Unnamed Canvas"}</span>
                            </Link>
                          );
                        })}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>

          <div className="justify-self-center">
            {topViewMode && onTopViewModeChange && (
              <div className="flex items-center rounded-md border border-gray-300 p-0.5">
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("canvas")}
                  className={`rounded px-2 py-1 text-xs font-medium ${
                    topViewMode === "canvas" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  Canvas
                </button>
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("yaml")}
                  className={`rounded px-2 py-1 text-xs font-medium ${
                    topViewMode === "yaml" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  YAML
                </button>
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("memory")}
                  className={`rounded px-2 py-1 text-xs font-medium ${
                    topViewMode === "memory" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  <span className="inline-flex items-center gap-1">
                    <span>Memory</span>
                    {memoryItemCount && memoryItemCount > 0 ? (
                      <span aria-label={`${memoryItemCount} memory items`}>({memoryItemCount})</span>
                    ) : null}
                  </span>
                </button>
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("settings")}
                  className={`rounded px-2 py-1 text-xs font-medium ${
                    topViewMode === "settings" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  Settings
                </button>
                {showVersioningTab ? (
                  <button
                    type="button"
                    onClick={() => onTopViewModeChange("versioning")}
                    className={`rounded px-2 py-1 text-xs font-medium ${
                      topViewMode === "versioning" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                    }`}
                  >
                    <span className="inline-flex items-center gap-1">
                      <span>Versioning</span>
                      {versioningItemCount && versioningItemCount > 0 ? (
                        <span aria-label={`${versioningItemCount} open change requests`}>({versioningItemCount})</span>
                      ) : null}
                    </span>
                  </button>
                ) : null}
              </div>
            )}
          </div>

          <div className="flex items-center gap-2 justify-self-end">
            {isDefaultMode ? (
              <>
                {onExportYamlCopy && onExportYamlDownload ? (
                  <DropdownMenu open={isYamlMenuOpen} onOpenChange={setIsYamlMenuOpen}>
                    <DropdownMenuTrigger asChild>
                      <Button variant="outline" size="sm" className="h-8 px-2 text-xs font-mono">
                        .yaml
                        <ChevronDown className="h-3.5 w-3.5" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-44 p-2">
                      <Button
                        type="button"
                        variant="ghost"
                        className="w-full justify-start"
                        size="sm"
                        onClick={() => {
                          onExportYamlCopy();
                          setIsYamlMenuOpen(false);
                        }}
                      >
                        Copy to clipboard
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        className="w-full justify-start"
                        size="sm"
                        onClick={() => {
                          onExportYamlDownload();
                          setIsYamlMenuOpen(false);
                        }}
                      >
                        Download file
                      </Button>
                    </DropdownMenuContent>
                  </DropdownMenu>
                ) : null}
                {unsavedMessage ? (
                  <span className="text-xs font-medium text-yellow-700 bg-orange-100 px-2 py-1 rounded hidden sm:inline">
                    {unsavedMessage}
                  </span>
                ) : null}
                {onToggleAutoSave
                  ? wrapWithTooltip(
                      autoSaveDisabled,
                      autoSaveDisabledTooltip,
                      <div className="flex items-center gap-2">
                        <label
                          htmlFor="auto-save-toggle"
                          className={`text-sm hidden sm:inline ${autoSaveDisabled ? "text-gray-400" : "text-gray-800"}`}
                        >
                          Auto-save
                        </label>
                        <Switch
                          id="auto-save-toggle"
                          checked={isAutoSaveEnabled}
                          onCheckedChange={(checked) => {
                            if (checked) {
                              onSave?.();
                            }
                            onToggleAutoSave?.();
                          }}
                          disabled={autoSaveDisabled}
                        />
                      </div>,
                    )
                  : null}
                {onUndo && canUndo ? (
                  <Button onClick={onUndo} size="sm" variant="outline">
                    <Undo2 />
                    Revert
                  </Button>
                ) : null}
                {onSave && !saveButtonHidden
                  ? wrapWithTooltip(
                      saveDisabled,
                      saveDisabledTooltip,
                      <Button
                        onClick={onSave}
                        size="sm"
                        variant={saveIsPrimary ? "default" : "outline"}
                        data-testid="save-canvas-button"
                        disabled={saveDisabled}
                      >
                        Save
                      </Button>,
                    )
                  : null}
              </>
            ) : null}

            {showVersioningDisabledBadge ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="rounded border border-amber-300 bg-amber-100 px-2 py-1 text-[11px] font-semibold uppercase tracking-wide text-amber-900">
                    VERSIONING OFF
                  </span>
                </TooltipTrigger>
                <TooltipContent side="top">
                  {versioningDisabledTooltip || "Versioning is disabled. Enable canvas versioning in canvas settings."}
                </TooltipContent>
              </Tooltip>
            ) : null}

            {showEditingDropdown ? (
              <DropdownMenu open={isEditingMenuOpen} onOpenChange={setIsEditingMenuOpen}>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="h-8 gap-2">
                    <SquarePen className="h-4 w-4" />
                    Editing
                    <ChevronDown className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-33 p-2">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="w-full">
                        {wrapWithTooltip(
                          publishVersionDisabled,
                          publishVersionDisabledTooltip,
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="w-full justify-start"
                            onClick={() => {
                              onPublishVersion?.();
                              setIsEditingMenuOpen(false);
                            }}
                            disabled={publishVersionDisabled || !onPublishVersion}
                          >
                            <Rocket className="h-4 w-4" />
                            Publish
                          </Button>,
                        )}
                      </div>
                    </TooltipTrigger>
                    <TooltipContent side="left">
                      Create and publish a change request from your current draft.
                    </TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="w-full">
                        {wrapWithTooltip(
                          discardVersionDisabled,
                          discardVersionDisabledTooltip,
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="w-full justify-start"
                            onClick={() => {
                              onDiscardVersion?.();
                              setIsEditingMenuOpen(false);
                            }}
                            disabled={discardVersionDisabled || !onDiscardVersion}
                          >
                            <RotateCcw className="h-4 w-4" />
                            Discard
                          </Button>,
                        )}
                      </div>
                    </TooltipTrigger>
                    <TooltipContent side="left">
                      Discard draft changes and reset it to match the current live version.
                    </TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="w-full">
                        {wrapWithTooltip(
                          exitEditModeDisabled,
                          exitEditModeDisabledTooltip,
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="w-full justify-start"
                            onClick={() => {
                              onExitEditMode?.();
                              setIsEditingMenuOpen(false);
                            }}
                            disabled={exitEditModeDisabled}
                          >
                            <LogOut className="h-4 w-4" />
                            Exit
                          </Button>,
                        )}
                      </div>
                    </TooltipTrigger>
                    <TooltipContent side="left">Exit edit mode and return to the live version.</TooltipContent>
                  </Tooltip>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null}

            {showSaveDropdown ? (
              <DropdownMenu open={isSaveMenuOpen} onOpenChange={setIsSaveMenuOpen}>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="h-8 gap-2" disabled={saveDisabled}>
                    {saveStatusIcon}
                    <span>{saveStatusLabel}</span>
                    <ChevronDown className="h-4 w-4 text-slate-500" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-40 p-2">
                  <div className="flex items-center justify-center gap-3 rounded-md px-2 py-1.5">
                    <span className="text-sm font-medium text-slate-800">Auto-save</span>
                    {wrapWithTooltip(
                      autoSaveToggleDisabled,
                      autoSaveDisabledTooltip,
                      <Switch
                        checked={!!isAutoSaveEnabled}
                        onCheckedChange={(checked) => {
                          if (checked) {
                            onSave?.();
                          }
                          onToggleAutoSave?.();
                        }}
                        disabled={autoSaveToggleDisabled}
                      />,
                    )}
                  </div>

                  {isAutoSaveEnabled ? (
                    <p className="px-2 pb-2 text-xs text-slate-600 text-center">Changes are saved automatically.</p>
                  ) : null}

                  {showSaveUndoActions ? (
                    <div className="space-y-1 border-t border-slate-200 pt-2">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="w-full justify-start"
                        onClick={() => {
                          onSave?.();
                          setIsSaveMenuOpen(false);
                        }}
                        disabled={saveDisabled || !onSave}
                      >
                        <CloudUpload className="h-4 w-4" />
                        Save
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="w-full justify-start"
                        onClick={() => {
                          onUndo?.();
                          setIsSaveMenuOpen(false);
                        }}
                        disabled={!onUndo || !canUndo}
                      >
                        <Undo2 className="h-4 w-4" />
                        Undo
                      </Button>
                    </div>
                  ) : null}
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null}

            {showEditButton
              ? wrapWithTooltip(
                  enterEditModeDisabled,
                  enterEditModeDisabledTooltip,
                  <div className="flex items-center gap-2">
                    {showPendingDraftBadge ? (
                      <div className="flex items-center">
                        <span className="rounded border border-amber-300 bg-amber-100 px-2 py-1 text-xs font-medium text-amber-900">
                          Unpublished Changes
                        </span>
                        <span
                          aria-hidden="true"
                          className="h-0 w-0 border-y-[6px] border-y-transparent border-l-[9px] border-l-amber-300"
                        />
                      </div>
                    ) : null}
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={onEnterEditMode}
                      disabled={enterEditModeDisabled}
                      className="h-8"
                    >
                      <Pencil className="h-2 w-2" />
                      Edit
                    </Button>
                  </div>,
                )
              : null}
          </div>
        </div>
      </header>
    </>
  );
}

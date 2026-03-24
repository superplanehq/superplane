import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Copy, Download, Home, ChevronDown, LogOut, Palette, RotateCcw, Undo2, Pencil, Rocket } from "lucide-react";
import { Button } from "../button";
import { Button as UIButton } from "@/components/ui/button";
import { Switch } from "../switch";
import { useCanvases } from "@/hooks/useCanvasData";
import { Link, useParams } from "react-router-dom";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/ui/select";

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
  topViewMode?: "canvas" | "yaml" | "memory" | "settings";
  onTopViewModeChange?: (mode: "canvas" | "yaml" | "memory" | "settings") => void;
  onExportYamlCopy?: () => void;
  onExportYamlDownload?: () => void;
  memoryItemCount?: number;
  mode?: HeaderMode;
  saveState?: SaveState;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  /** When &gt; 0 (unpublished draft diff items), shown as "Propose Change (n)" in version edit mode. */
  unpublishedDraftChangeCount?: number;
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
  topViewMode,
  onTopViewModeChange,
  onExportYamlCopy,
  onExportYamlDownload,
  memoryItemCount,
  mode = "default",
  saveState: _saveState = "saved",
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  unpublishedDraftChangeCount = 0,
}: HeaderProps) {
  const { workflowId } = useParams<{ workflowId?: string }>();
  const { data: workflows = [], isLoading: workflowsLoading } = useCanvases(organizationId || "");
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [isYamlMenuOpen, setIsYamlMenuOpen] = useState(false);
  const [exportAction, setExportAction] = useState<string>("");
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

  const isVersioningDisabledMode = mode === "versioning-disabled";
  const isDefaultMode = mode === "default" || isVersioningDisabledMode;
  const showEditButton = mode === "version-live";
  const showVersionEditActions = mode === "version-edit";
  const proposeChangeLabel =
    unpublishedDraftChangeCount > 0 ? `Propose Change (${unpublishedDraftChangeCount})` : "Propose Change";

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
                  className={`rounded-sm px-2 py-1 text-xs font-medium ${
                    topViewMode === "canvas" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  Canvas
                </button>
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("yaml")}
                  className={`rounded-sm px-2 py-1 text-xs font-medium ${
                    topViewMode === "yaml" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  YAML
                </button>
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("memory")}
                  className={`rounded-sm px-2 py-1 text-xs font-medium ${
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
                  className={`rounded-sm px-2 py-1 text-xs font-medium ${
                    topViewMode === "settings" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  Settings
                </button>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2 justify-self-end">
            {isDefaultMode ? (
              <>
                {isVersioningDisabledMode && onExportYamlCopy && onExportYamlDownload ? (
                  <Select
                    value={exportAction || undefined}
                    onValueChange={(value) => {
                      setExportAction(value);
                      if (value === "copy") {
                        onExportYamlCopy();
                      }
                      if (value === "download") {
                        onExportYamlDownload();
                      }
                      setExportAction("");
                    }}
                  >
                    <SelectTrigger className="h-5 w-fit min-w-0 rounded-md border-gray-300 px-1 py-0 text-xs font-mono text-gray-500 data-[placeholder]:text-gray-500 shadow-none [&>svg]:hidden">
                      <SelectValue placeholder=".yaml" />
                    </SelectTrigger>
                    <SelectContent align="end">
                      <SelectItem value="copy">
                        <span className="flex items-center gap-2">
                          <Copy className="h-3.5 w-3.5" />
                          Copy to Clipboard
                        </span>
                      </SelectItem>
                      <SelectItem value="download">
                        <span className="flex items-center gap-2">
                          <Download className="h-3.5 w-3.5" />
                          Download File
                        </span>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                ) : null}
                {!isVersioningDisabledMode && onExportYamlCopy && onExportYamlDownload ? (
                  <DropdownMenu open={isYamlMenuOpen} onOpenChange={setIsYamlMenuOpen}>
                    <DropdownMenuTrigger asChild>
                      <Button variant="outline" size="sm" className="h-8 px-2 text-xs font-mono">
                        .yaml
                        <ChevronDown className="h-3.5 w-3.5" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="w-44 p-2">
                      <UIButton
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
                      </UIButton>
                      <UIButton
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
                      </UIButton>
                    </DropdownMenuContent>
                  </DropdownMenu>
                ) : null}
                {!isVersioningDisabledMode && unsavedMessage ? (
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
                          onCheckedChange={
                            isVersioningDisabledMode
                              ? onToggleAutoSave
                              : (checked) => {
                                  if (checked) {
                                    onSave?.();
                                  }
                                  onToggleAutoSave?.();
                                }
                          }
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

            {showVersionEditActions ? (
              <div className="flex items-center gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <UIButton
                        type="button"
                        variant="outline"
                        size="icon-sm"
                        className="h-8 shrink-0"
                        onClick={() => onDiscardVersion?.()}
                        disabled={discardVersionDisabled || !onDiscardVersion}
                        aria-label="Discard draft"
                      >
                        <RotateCcw className="h-4 w-4" />
                      </UIButton>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom">
                    {discardVersionDisabled && discardVersionDisabledTooltip
                      ? discardVersionDisabledTooltip
                      : "Discard draft changes and reset to the current live version."}
                  </TooltipContent>
                </Tooltip>
                {wrapWithTooltip(
                  publishVersionDisabled,
                  publishVersionDisabledTooltip,
                  <UIButton
                    type="button"
                    variant="default"
                    size="sm"
                    className="h-8 gap-1.5"
                    onClick={() => onPublishVersion?.()}
                    disabled={publishVersionDisabled || !onPublishVersion}
                  >
                    <Rocket className="h-4 w-4" />
                    {proposeChangeLabel}
                  </UIButton>,
                )}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <UIButton
                        type="button"
                        variant="outline"
                        size="icon-sm"
                        className="h-8 shrink-0"
                        onClick={() => onExitEditMode?.()}
                        disabled={exitEditModeDisabled}
                        aria-label="Exit edit mode"
                      >
                        <LogOut className="h-4 w-4" />
                      </UIButton>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent side="bottom">
                    {exitEditModeDisabled && exitEditModeDisabledTooltip
                      ? exitEditModeDisabledTooltip
                      : "Exit edit mode and return to the live version."}
                  </TooltipContent>
                </Tooltip>
              </div>
            ) : null}

            {showEditButton
              ? wrapWithTooltip(
                  enterEditModeDisabled,
                  enterEditModeDisabledTooltip,
                  <div className="flex items-center gap-2">
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

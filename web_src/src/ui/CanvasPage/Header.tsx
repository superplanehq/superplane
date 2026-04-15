import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { ChevronDown, MoreVertical, RotateCcw, Pencil, Settings } from "lucide-react";
import { Button } from "../button";
import { Button as UIButton } from "@/components/ui/button";
import { useCanvases } from "@/hooks/useCanvasData";
import { useNavigate, useParams } from "react-router-dom";
import { useState, type ReactNode } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

export interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
  href?: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
}

type HeaderMode = "default" | "version-live" | "version-edit";

type CanvasTopViewTab = "canvas" | "yaml" | "cli" | "memory";

interface HeaderProps {
  breadcrumbs: BreadcrumbItem[];
  onSave?: () => void;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  onLogoClick?: () => void;
  organizationId?: string;
  unsavedMessage?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  topViewMode?: CanvasTopViewTab;
  onTopViewModeChange?: (mode: CanvasTopViewTab) => void;
  onExportYamlCopy?: () => void;
  onExportYamlDownload?: () => void;
  memoryItemCount?: number;
  mode?: HeaderMode;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  /** Label for the publish/propose-change button in version edit mode. Defaults to "Publish". */
  publishVersionLabel?: string;
  /** When &gt; 0 (unpublished draft diff items), shown as badge count on the publish button in version edit mode. */
  unpublishedDraftChangeCount?: number;
  /** Canvas settings route requires `canvases:update`; hide the menu when the user cannot update. */
  showCanvasSettingsMenu?: boolean;
}

export function Header({
  breadcrumbs,
  onSave,
  onPublishVersion,
  onDiscardVersion,
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
  topViewMode,
  onTopViewModeChange,
  onExportYamlCopy,
  onExportYamlDownload,
  memoryItemCount,
  mode = "default",
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  publishVersionLabel = "Publish",
  unpublishedDraftChangeCount = 0,
  showCanvasSettingsMenu = true,
}: HeaderProps) {
  const navigate = useNavigate();
  const { workflowId, canvasId: canvasIdParam } = useParams<{ workflowId?: string; canvasId?: string }>();
  const activeCanvasId = canvasIdParam || workflowId;
  const { data: workflows = [], isLoading: workflowsLoading } = useCanvases(organizationId || "");
  const [isYamlMenuOpen, setIsYamlMenuOpen] = useState(false);

  const currentWorkflowName = (() => {
    if (activeCanvasId) {
      const workflow = workflows.find((w) => w.metadata?.id === activeCanvasId);
      if (workflow?.metadata?.name) {
        return workflow.metadata.name;
      }
    }
    if (breadcrumbs.length > 1 && breadcrumbs[1]?.label) {
      return breadcrumbs[1].label;
    }
    return breadcrumbs.length > 0 ? breadcrumbs[breadcrumbs.length - 1].label : "";
  })();

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
  const showVersionEditActions = mode === "version-edit";
  const hasChanges = unpublishedDraftChangeCount > 0;
  const publishButtonLabel = hasChanges
    ? `${publishVersionLabel} (${unpublishedDraftChangeCount})`
    : publishVersionLabel;

  const showSecondaryHeaderRow = true;

  return (
    <header className="border-b border-slate-950/15 bg-white">
      {/* Top bar: nav + title + canvas menu */}
      <div className="relative flex h-11 items-center border-b border-slate-950/15 px-3 sm:px-4">
        <div className="relative z-10 flex min-w-0 shrink-0 items-center">
          <OrganizationMenuButton organizationId={organizationId} onLogoClick={onLogoClick} />
        </div>
        <div className="pointer-events-none absolute inset-x-0 flex justify-center px-24">
          <span className="truncate text-center text-sm font-medium text-slate-900">
            {currentWorkflowName || (workflowsLoading ? "Loading…" : "Canvas")}
          </span>
        </div>
        <div className="relative z-10 ml-auto flex shrink-0 items-center">
          {showCanvasSettingsMenu && organizationId && activeCanvasId ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <UIButton
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 text-slate-600"
                  aria-label="Canvas menu"
                >
                  <MoreVertical className="h-4 w-4" />
                </UIButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem onClick={() => navigate(`/${organizationId}/canvases/${activeCanvasId}/settings`)}>
                  <Settings className="h-4 w-4" />
                  Settings
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </div>
      </div>

      {showSecondaryHeaderRow ? (
        <div className="relative grid h-12 grid-cols-3 items-center px-4">
          <div className="min-w-0 justify-self-start" aria-hidden />

          <div className="justify-self-center">
            {topViewMode && onTopViewModeChange && (
              <div className="flex items-center rounded-md border border-slate-950/15 p-0.5 text-[13px] font-medium">
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("canvas")}
                  className={`rounded-sm px-2 py-0.5 ${
                    topViewMode === "canvas" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  Canvas
                </button>
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("cli")}
                  className={`rounded-sm px-2 py-0.5 ${
                    topViewMode === "cli" ? "bg-slate-900 text-white" : "text-gray-700 hover:bg-gray-100"
                  }`}
                >
                  CLI
                </button>
                <button
                  type="button"
                  onClick={() => onTopViewModeChange("memory")}
                  className={`rounded-sm px-2 py-0.5 ${
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
                {unsavedMessage ? (
                  <span className="hidden rounded bg-orange-100 px-2 py-1 text-xs font-medium text-yellow-700 sm:inline">
                    {unsavedMessage}
                  </span>
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
                {hasChanges ? (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="inline-flex">
                        <UIButton
                          type="button"
                          variant="outline"
                          size="icon-xs"
                          className="shrink-0"
                          onClick={() => onDiscardVersion?.()}
                          disabled={discardVersionDisabled || !onDiscardVersion}
                          aria-label="Discard draft"
                        >
                          <RotateCcw className="h-3.5 w-3.5" />
                        </UIButton>
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="bottom">
                      {discardVersionDisabled && discardVersionDisabledTooltip
                        ? discardVersionDisabledTooltip
                        : "Discard draft changes and reset to the current live version."}
                    </TooltipContent>
                  </Tooltip>
                ) : null}
                {wrapWithTooltip(
                  publishVersionDisabled,
                  publishVersionDisabledTooltip,
                  <UIButton
                    type="button"
                    variant="default"
                    size="sm"
                    onClick={() => onPublishVersion?.()}
                    disabled={publishVersionDisabled || !onPublishVersion}
                  >
                    {publishButtonLabel}
                  </UIButton>,
                )}
              </div>
            ) : null}

            {showEditButton
              ? wrapWithTooltip(
                  enterEditModeDisabled,
                  enterEditModeDisabledTooltip,
                  <UIButton
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={onEnterEditMode}
                    disabled={enterEditModeDisabled}
                  >
                    <Pencil className="size-3.5" />
                    Edit
                  </UIButton>,
                )
              : null}
          </div>
        </div>
      ) : null}
    </header>
  );
}

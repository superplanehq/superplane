import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Button as UIButton } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreVertical, RotateCcw, Settings } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../button";

type HeaderMode = "default" | "version-live" | "version-edit";

interface HeaderProps {
  /** Shown centered in the top bar (canvas or template display name). */
  canvasName: string;
  onSave?: () => void;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  onLogoClick?: () => void;
  organizationId?: string;
  saveIsPrimary?: boolean;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  mode?: HeaderMode;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  /** Label for the publish/propose-change button in version edit mode. Defaults to "Publish". */
  publishVersionLabel?: string;
  /** When &gt; 0 (unpublished draft diff items), shown as badge count on the publish button in version edit mode. */
  unpublishedDraftChangeCount?: number;
  /** Canvas settings route requires `canvases:update`; hide the menu when the user cannot update. */
  showCanvasSettingsMenu?: boolean;
}

export function Header({
  canvasName,
  onSave,
  onPublishVersion,
  onDiscardVersion,
  onLogoClick,
  organizationId,
  saveIsPrimary,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  mode = "default",
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  publishVersionLabel = "Publish",
  unpublishedDraftChangeCount = 0,
  showCanvasSettingsMenu = true,
}: HeaderProps) {
  const headerTitle = canvasName.trim() || "Canvas";

  const isDefaultMode = mode === "default";
  const showVersionEditActions = mode === "version-edit";
  const hasChanges = unpublishedDraftChangeCount > 0;
  const publishButtonLabel = hasChanges
    ? `${publishVersionLabel} (${unpublishedDraftChangeCount})`
    : publishVersionLabel;

  return (
    <header>
      <PageHeader
        organizationId={organizationId}
        onLogoClick={onLogoClick}
        headerTitle={headerTitle}
        showCanvasSettingsMenu={showCanvasSettingsMenu}
      />

      <SecondaryHeader
        isDefaultMode={isDefaultMode}
        onSave={onSave}
        saveButtonHidden={saveButtonHidden}
        saveDisabled={saveDisabled}
        saveDisabledTooltip={saveDisabledTooltip}
        saveIsPrimary={saveIsPrimary}
        headerMode={mode}
        showVersionEditActions={showVersionEditActions}
        hasChanges={hasChanges}
        onDiscardVersion={onDiscardVersion}
        discardVersionDisabled={discardVersionDisabled}
        discardVersionDisabledTooltip={discardVersionDisabledTooltip}
        publishVersionDisabled={publishVersionDisabled}
        publishVersionDisabledTooltip={publishVersionDisabledTooltip}
        onPublishVersion={onPublishVersion}
        publishButtonLabel={publishButtonLabel}
        onEnterEditMode={onEnterEditMode}
        enterEditModeDisabled={enterEditModeDisabled}
        enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
        onExitEditMode={onExitEditMode}
        exitEditModeDisabled={exitEditModeDisabled}
        exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
      />
    </header>
  );
}

function PageHeader({
  organizationId,
  onLogoClick,
  headerTitle,
  showCanvasSettingsMenu = true,
}: {
  organizationId?: string;
  onLogoClick?: () => void;
  headerTitle: string;
  showCanvasSettingsMenu?: boolean;
}) {
  const navigate = useNavigate();
  const { workflowId, canvasId: canvasIdParam } = useParams<{ workflowId?: string; canvasId?: string }>();
  const activeCanvasId = canvasIdParam || workflowId;

  return (
    <div className="relative flex h-11 items-center border-b border-slate-950/15 px-3 sm:px-4">
      <div className="relative z-10 flex min-w-0 shrink-0 items-center">
        <OrganizationMenuButton organizationId={organizationId} onLogoClick={onLogoClick} />
      </div>
      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-24">
        <span className="truncate text-center text-sm font-medium text-slate-900">{headerTitle}</span>
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
  );
}

function SecondaryHeader({
  isDefaultMode,
  onSave,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  saveIsPrimary,
  headerMode,
  showVersionEditActions,
  hasChanges,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  onPublishVersion,
  publishButtonLabel,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
}: {
  isDefaultMode: boolean;
  onSave?: () => void;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  saveIsPrimary?: boolean;
  headerMode: HeaderMode;
  showVersionEditActions: boolean;
  hasChanges: boolean;
  onDiscardVersion?: () => void;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  onPublishVersion?: () => void;
  publishButtonLabel: string;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
}) {
  const showCanvasViewModeToggle = headerMode === "version-live" || headerMode === "version-edit";
  const canvasViewMode = headerMode === "version-edit" ? "version-edit" : "version-live";

  return (
    <div className="relative flex h-12 items-center border-b border-slate-950/15 bg-slate-100 px-4">
      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-16 sm:px-24">
        <div className="pointer-events-auto">
          {showCanvasViewModeToggle && onEnterEditMode && onExitEditMode ? (
            <CanvasViewModeToggle
              mode={canvasViewMode}
              onSelectEditor={onEnterEditMode}
              onSelectLive={onExitEditMode}
              enterEditorDisabled={!!enterEditModeDisabled}
              enterEditorDisabledTooltip={enterEditModeDisabledTooltip}
              exitEditorDisabled={!!exitEditModeDisabled}
              exitEditorDisabledTooltip={exitEditModeDisabledTooltip}
            />
          ) : null}
        </div>
      </div>

      <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
        {isDefaultMode && onSave && !saveButtonHidden ? (
          <SaveButton
            onSave={onSave}
            saveDisabled={saveDisabled}
            saveDisabledTooltip={saveDisabledTooltip}
            saveIsPrimary={saveIsPrimary}
          />
        ) : null}

        {showVersionEditActions ? (
          <div className="flex items-center gap-2">
            {hasChanges ? (
              <DiscardDraftButton
                onDiscard={() => onDiscardVersion?.()}
                disabled={discardVersionDisabled || !onDiscardVersion}
                disabledTooltip={discardVersionDisabledTooltip}
              />
            ) : null}
            <PublishVersionButton
              onPublish={() => onPublishVersion?.()}
              label={publishButtonLabel}
              disabled={publishVersionDisabled || !onPublishVersion}
              publishVersionDisabled={!!publishVersionDisabled}
              publishVersionDisabledTooltip={publishVersionDisabledTooltip}
            />
          </div>
        ) : null}
      </div>
    </div>
  );
}

function SaveButton({
  onSave,
  saveDisabled,
  saveDisabledTooltip,
  saveIsPrimary,
}: {
  onSave: () => void;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  saveIsPrimary?: boolean;
}) {
  if (saveDisabled && saveDisabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">
            <Button
              onClick={onSave}
              size="sm"
              variant={saveIsPrimary ? "default" : "outline"}
              data-testid="save-canvas-button"
              disabled={saveDisabled}
            >
              Save
            </Button>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">{saveDisabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <Button
      onClick={onSave}
      size="sm"
      variant={saveIsPrimary ? "default" : "outline"}
      data-testid="save-canvas-button"
      disabled={saveDisabled}
    >
      Save
    </Button>
  );
}

function DiscardDraftButton({
  onDiscard,
  disabled,
  disabledTooltip,
}: {
  onDiscard: () => void;
  disabled: boolean;
  disabledTooltip?: string;
}) {
  const tooltipText =
    disabled && disabledTooltip ? disabledTooltip : "Discard draft changes and reset to the current live version.";

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex">
          <UIButton
            type="button"
            variant="outline"
            size="icon-xs"
            className="shrink-0"
            onClick={onDiscard}
            disabled={disabled}
            aria-label="Discard draft"
          >
            <RotateCcw className="h-3.5 w-3.5" />
          </UIButton>
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom">{tooltipText}</TooltipContent>
    </Tooltip>
  );
}

function PublishVersionButton({
  onPublish,
  label,
  disabled,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
}: {
  onPublish: () => void;
  label: string;
  disabled: boolean;
  publishVersionDisabled: boolean;
  publishVersionDisabledTooltip?: string;
}) {
  if (publishVersionDisabled && publishVersionDisabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">
            <UIButton type="button" variant="default" size="sm" onClick={onPublish} disabled={disabled}>
              {label}
            </UIButton>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">{publishVersionDisabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <UIButton type="button" variant="default" size="sm" onClick={onPublish} disabled={disabled}>
      {label}
    </UIButton>
  );
}

type CanvasViewMode = "version-live" | "version-edit";

function CanvasViewModeToggle({
  mode,
  onSelectEditor,
  onSelectLive,
  enterEditorDisabled,
  enterEditorDisabledTooltip,
  exitEditorDisabled,
  exitEditorDisabledTooltip,
}: {
  mode: CanvasViewMode;
  onSelectEditor: () => void;
  onSelectLive: () => void;
  enterEditorDisabled: boolean;
  enterEditorDisabledTooltip?: string;
  exitEditorDisabled: boolean;
  exitEditorDisabledTooltip?: string;
}) {
  const handleValueChange = (next: string) => {
    if (next === "version-edit" && mode === "version-live") {
      void onSelectEditor();
    } else if (next === "version-live" && mode === "version-edit") {
      void onSelectLive();
    }
  };

  const editorDisabled = mode === "version-live" && enterEditorDisabled;
  const liveDisabled = mode === "version-edit" && exitEditorDisabled;

  const editorTrigger = (
    <TabsTrigger
      value="version-edit"
      disabled={editorDisabled}
      data-testid="canvas-view-mode-editor"
      aria-label="Editor"
    >
      Editor
    </TabsTrigger>
  );

  const liveTrigger = (
    <TabsTrigger
      value="version-live"
      disabled={liveDisabled}
      data-testid="canvas-view-mode-live"
      aria-label="Live Canvas"
    >
      Live Canvas
    </TabsTrigger>
  );

  return (
    <Tabs value={mode} onValueChange={handleValueChange} className="inline-flex w-auto" aria-label="Canvas view">
      <TabsList className="h-8 w-fit gap-0">
        {editorDisabled && enterEditorDisabledTooltip ? (
          <Tooltip>
            <TooltipTrigger asChild>{editorTrigger}</TooltipTrigger>
            <TooltipContent side="top">{enterEditorDisabledTooltip}</TooltipContent>
          </Tooltip>
        ) : (
          editorTrigger
        )}
        {liveDisabled && exitEditorDisabledTooltip ? (
          <Tooltip>
            <TooltipTrigger asChild>{liveTrigger}</TooltipTrigger>
            <TooltipContent side="top">{exitEditorDisabledTooltip}</TooltipContent>
          </Tooltip>
        ) : (
          liveTrigger
        )}
      </TabsList>
    </Tabs>
  );
}

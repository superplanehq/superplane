import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreVertical, Settings } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../button";
import { CanvasModeToggle } from "./components/CanvasModeToggle";

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
  onExitEditMode,
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
            <CanvasModeToggle mode={canvasViewMode} onSelectEditor={onEnterEditMode} onSelectLive={onExitEditMode} />
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
  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">
            <UIButton type="button" variant="outline" size="sm" onClick={onDiscard} disabled={disabled}>
              Discard
            </UIButton>
          </div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <UIButton type="button" variant="outline" size="sm" onClick={onDiscard} disabled={disabled}>
      Discard
    </UIButton>
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

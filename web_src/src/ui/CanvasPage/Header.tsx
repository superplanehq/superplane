import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { MoreVertical, RotateCcw, Pencil, Settings } from "lucide-react";
import { Button } from "../button";
import { Button as UIButton } from "@/components/ui/button";
import { useNavigate, useParams } from "react-router-dom";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";

type HeaderMode = "default" | "version-live" | "version-edit";

interface HeaderProps {
  /** Shown centered in the top bar (canvas or template display name). */
  canvasName: string;
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
  canvasName,
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
  mode = "default",
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  publishVersionLabel = "Publish",
  unpublishedDraftChangeCount = 0,
  showCanvasSettingsMenu = true,
}: HeaderProps) {
  const headerTitle = canvasName.trim() || "Canvas";

  const isDefaultMode = mode === "default";
  const showEditButton = mode === "version-live";
  const showVersionEditActions = mode === "version-edit";
  const hasChanges = unpublishedDraftChangeCount > 0;
  const publishButtonLabel = hasChanges
    ? `${publishVersionLabel} (${unpublishedDraftChangeCount})`
    : publishVersionLabel;

  return (
    <header className="border-b border-slate-950/15 bg-white">
      <PageHeader
        organizationId={organizationId}
        onLogoClick={onLogoClick}
        headerTitle={headerTitle}
        showCanvasSettingsMenu={showCanvasSettingsMenu}
      />

      <SecondaryHeader
        isDefaultMode={isDefaultMode}
        unsavedMessage={unsavedMessage}
        onSave={onSave}
        saveButtonHidden={saveButtonHidden}
        saveDisabled={saveDisabled}
        saveDisabledTooltip={saveDisabledTooltip}
        saveIsPrimary={saveIsPrimary}
        showVersionEditActions={showVersionEditActions}
        hasChanges={hasChanges}
        onDiscardVersion={onDiscardVersion}
        discardVersionDisabled={discardVersionDisabled}
        discardVersionDisabledTooltip={discardVersionDisabledTooltip}
        publishVersionDisabled={publishVersionDisabled}
        publishVersionDisabledTooltip={publishVersionDisabledTooltip}
        onPublishVersion={onPublishVersion}
        publishButtonLabel={publishButtonLabel}
        showEditButton={showEditButton}
        enterEditModeDisabled={enterEditModeDisabled}
        enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
        onEnterEditMode={onEnterEditMode}
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
  unsavedMessage,
  onSave,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  saveIsPrimary,
  showVersionEditActions,
  hasChanges,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
  onPublishVersion,
  publishButtonLabel,
  showEditButton,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onEnterEditMode,
}: {
  isDefaultMode: boolean;
  unsavedMessage?: string;
  onSave?: () => void;
  saveButtonHidden?: boolean;
  saveDisabled?: boolean;
  saveDisabledTooltip?: string;
  saveIsPrimary?: boolean;
  showVersionEditActions: boolean;
  hasChanges: boolean;
  onDiscardVersion?: () => void;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  onPublishVersion?: () => void;
  publishButtonLabel: string;
  showEditButton: boolean;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onEnterEditMode?: () => void;
}) {
  return (
    <div className="relative flex h-12 items-center justify-end gap-2 px-4">
      {isDefaultMode ? (
        <>
          {unsavedMessage ? (
            <span className="hidden rounded bg-orange-100 px-2 py-1 text-xs font-medium text-yellow-700 sm:inline">
              {unsavedMessage}
            </span>
          ) : null}
          {onSave && !saveButtonHidden ? (
            saveDisabled && saveDisabledTooltip ? (
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
            ) : (
              <Button
                onClick={onSave}
                size="sm"
                variant={saveIsPrimary ? "default" : "outline"}
                data-testid="save-canvas-button"
                disabled={saveDisabled}
              >
                Save
              </Button>
            )
          ) : null}
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
          {publishVersionDisabled && publishVersionDisabledTooltip ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="inline-flex">
                  <UIButton
                    type="button"
                    variant="default"
                    size="sm"
                    onClick={() => onPublishVersion?.()}
                    disabled={publishVersionDisabled || !onPublishVersion}
                  >
                    {publishButtonLabel}
                  </UIButton>
                </div>
              </TooltipTrigger>
              <TooltipContent side="top">{publishVersionDisabledTooltip}</TooltipContent>
            </Tooltip>
          ) : (
            <UIButton
              type="button"
              variant="default"
              size="sm"
              onClick={() => onPublishVersion?.()}
              disabled={publishVersionDisabled || !onPublishVersion}
            >
              {publishButtonLabel}
            </UIButton>
          )}
        </div>
      ) : null}

      {showEditButton ? (
        enterEditModeDisabled && enterEditModeDisabledTooltip ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="inline-flex">
                <UIButton
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={onEnterEditMode}
                  disabled={enterEditModeDisabled}
                >
                  <Pencil className="size-3.5" />
                  Edit
                </UIButton>
              </div>
            </TooltipTrigger>
            <TooltipContent side="top">{enterEditModeDisabledTooltip}</TooltipContent>
          </Tooltip>
        ) : (
          <UIButton
            type="button"
            variant="outline"
            size="sm"
            onClick={onEnterEditMode}
            disabled={enterEditModeDisabled}
          >
            <Pencil className="size-3.5" />
            Edit
          </UIButton>
        )
      ) : null}
    </div>
  );
}

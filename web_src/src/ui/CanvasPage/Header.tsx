import type { AgentState } from "@/components/AgentSidebar/useAgentState";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { ArrowLeft, History, MoreVertical, Settings } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../button";
import { AgentSidebarTrigger } from "./components/AgentSidebarTrigger";
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
  /** When true, shows the Discard control next to Publish in version edit mode (draft differs from live). */
  hasUnpublishedDraftChanges?: boolean;
  /** Canvas settings route requires `canvases:update`; hide the menu when the user cannot update. */
  showCanvasSettingsMenu?: boolean;
  isVersionControlOpen?: boolean;
  /** Opens the version history sidebar (not a toggle). */
  onOpenVersionControl?: () => void;
  /** Closes the version history sidebar (e.g. "Exit Version History" in the secondary header). */
  onCloseVersionControl?: () => void;
  versionControlButtonTooltip?: string;
  versionControlNotificationCount?: number;
  agentState: AgentState;
}

export function Header(props: HeaderProps) {
  const headerTitle = props.canvasName.trim() || "Canvas";

  return (
    <header>
      <PageHeader
        organizationId={props.organizationId}
        onLogoClick={props.onLogoClick}
        headerTitle={headerTitle}
        showCanvasSettingsMenu={props.showCanvasSettingsMenu}
      />

      <SecondaryHeader {...props} />
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
    <div className="relative z-40 flex h-11 items-center border-b border-slate-950/15 px-3 sm:px-4">
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

function SecondaryHeader(props: HeaderProps) {
  const showCanvasViewModeToggle = props.mode === "version-live" || props.mode === "version-edit";
  const canvasViewMode = props.mode === "version-edit" ? "version-edit" : "version-live";
  const showVersionHistoryExit = props.isVersionControlOpen && !!props.onCloseVersionControl;

  if (showVersionHistoryExit) {
    return (
      <div className="relative z-10 flex h-11 items-center justify-start border-b border-border bg-slate-100 px-4">
        <button
          type="button"
          onClick={props.onCloseVersionControl}
          className="inline-flex items-center gap-1.5 text-sm font-medium text-slate-600 transition-colors hover:text-slate-800"
          aria-label="Exit version history"
        >
          <ArrowLeft className="h-4 w-4 shrink-0" aria-hidden />
          Exit Version History
        </button>
      </div>
    );
  }

  return (
    <div className="relative z-10 flex h-11 items-center border-b border-border bg-slate-100 px-4 gap-3">
      <AgentSidebarTrigger agentState={props.agentState} />

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-16 sm:px-24">
        <div className="pointer-events-auto">
          {showCanvasViewModeToggle && props.onEnterEditMode && props.onExitEditMode ? (
            <CanvasModeToggle
              mode={canvasViewMode}
              onSelectEditor={props.onEnterEditMode}
              onSelectLive={props.onExitEditMode}
            />
          ) : null}
        </div>
      </div>

      <SecondaryHeaderActions {...props} />
    </div>
  );
}

function SecondaryHeaderActions({
  mode,
  onOpenVersionControl,
  versionControlButtonTooltip,
  versionControlNotificationCount = 0,
  onSave,
  saveButtonHidden,
  saveDisabled,
  saveDisabledTooltip,
  saveIsPrimary,
  hasUnpublishedDraftChanges,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  onPublishVersion,
  publishVersionLabel,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
}: HeaderProps) {
  const showVersionControlTrigger = !!onOpenVersionControl;

  return (
    <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
      {mode === "default" && onSave && !saveButtonHidden ? (
        <SaveButton
          onSave={onSave}
          saveDisabled={saveDisabled}
          saveDisabledTooltip={saveDisabledTooltip}
          saveIsPrimary={saveIsPrimary}
        />
      ) : null}

      {mode === "version-edit" ? (
        <div className="flex items-center gap-2">
          {showVersionControlTrigger ? (
            <VersionControlButton
              onOpen={onOpenVersionControl}
              tooltip={versionControlButtonTooltip}
              notificationCount={versionControlNotificationCount}
            />
          ) : null}
          {hasUnpublishedDraftChanges ? (
            <DiscardDraftButton
              onDiscard={() => onDiscardVersion?.()}
              disabled={discardVersionDisabled || !onDiscardVersion}
              disabledTooltip={discardVersionDisabledTooltip}
            />
          ) : null}
          <PublishVersionButton
            onPublish={() => onPublishVersion?.()}
            label={publishVersionLabel || "Publish"}
            disabled={publishVersionDisabled || !onPublishVersion}
            publishVersionDisabled={!!publishVersionDisabled}
            publishVersionDisabledTooltip={publishVersionDisabledTooltip}
          />
        </div>
      ) : null}
    </div>
  );
}

function VersionControlButton({
  onOpen,
  tooltip,
  notificationCount,
}: {
  onOpen?: () => void;
  tooltip?: string;
  notificationCount: number;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="relative inline-flex">
          <UIButton type="button" variant="outline" size="icon-xs" onClick={onOpen} aria-label="Open version history">
            <History />
          </UIButton>
          {notificationCount > 0 ? (
            <span className="absolute left-4 -top-1.5 inline-flex min-w-[1.125rem] items-center justify-center rounded-full bg-orange-600 px-1 text-[10px] font-semibold leading-4 text-white">
              {notificationCount > 99 ? "99+" : notificationCount}
            </span>
          ) : null}
        </span>
      </TooltipTrigger>
      <TooltipContent side="top">{tooltip || "View version history"}</TooltipContent>
    </Tooltip>
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

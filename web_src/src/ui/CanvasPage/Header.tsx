import type { AgentState } from "@/components/AgentSidebar/useAgentState";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { GitBranch, MoreVertical, Pause, Play, Settings } from "lucide-react";
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
  onOpenVersionControl?: () => void;
  versionControlButtonTooltip?: string;
  versionControlNotificationCount?: number;
  paused?: boolean;
  onToggleCanvasPause?: (paused: boolean) => void;
  canvasPauseDisabled?: boolean;
  canvasPauseDisabledTooltip?: string;
  agentState: AgentState;
}

export function Header(props: HeaderProps) {
  const headerTitle = props.canvasName.trim() || "Canvas";

  return (
    <header>
      <PageHeader
        organizationId={props.organizationId}
        headerTitle={headerTitle}
        showCanvasSettingsMenu={props.showCanvasSettingsMenu}
      />

      <SecondaryHeader {...props} />
    </header>
  );
}

function PageHeader({
  organizationId,
  headerTitle,
  showCanvasSettingsMenu = true,
}: {
  organizationId?: string;
  headerTitle: string;
  showCanvasSettingsMenu?: boolean;
}) {
  const navigate = useNavigate();
  const { workflowId, canvasId: canvasIdParam } = useParams<{ workflowId?: string; canvasId?: string }>();
  const activeCanvasId = canvasIdParam || workflowId;

  return (
    <div className="relative flex h-11 items-center border-b border-slate-950/15 px-3 sm:px-4">
      <div className="relative z-10 flex min-w-0 shrink-0 items-center">
        <OrganizationMenuButton organizationId={organizationId} />
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

  return (
    <div className="relative flex h-12 items-center border-b border-slate-950/15 bg-slate-100 px-4 gap-3">
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
  isVersionControlOpen,
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
  paused,
  onToggleCanvasPause,
  canvasPauseDisabled,
  canvasPauseDisabledTooltip,
}: HeaderProps) {
  const showVersionControlTrigger = mode === "version-live" && !!onOpenVersionControl;

  return (
    <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
      {mode === "version-live" && onToggleCanvasPause ? (
        <PauseCanvasButton
          paused={!!paused}
          onToggle={onToggleCanvasPause}
          disabled={canvasPauseDisabled}
          disabledTooltip={canvasPauseDisabledTooltip}
        />
      ) : null}

      {showVersionControlTrigger ? (
        <VersionControlButton
          onToggle={onOpenVersionControl}
          isOpen={!!isVersionControlOpen}
          tooltip={versionControlButtonTooltip}
          notificationCount={versionControlNotificationCount}
        />
      ) : null}

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
  onToggle,
  isOpen,
  tooltip,
  notificationCount,
}: {
  onToggle: () => void;
  isOpen: boolean;
  tooltip?: string;
  notificationCount: number;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="relative inline-flex">
          <UIButton
            type="button"
            variant="outline"
            size="icon"
            className={isOpen ? "h-8 w-8 bg-slate-200 border-slate-300" : "h-8 w-8"}
            onClick={onToggle}
            aria-label={isOpen ? "Close version control" : "Open version control"}
            aria-pressed={isOpen}
          >
            <GitBranch className="h-4 w-4" />
          </UIButton>
          {notificationCount > 0 ? (
            <span className="absolute left-5 -top-1.5 inline-flex min-w-[1.125rem] items-center justify-center rounded-full bg-orange-600 px-1 text-[10px] font-semibold leading-4 text-white">
              {notificationCount > 99 ? "99+" : notificationCount}
            </span>
          ) : null}
        </span>
      </TooltipTrigger>
      <TooltipContent side="top">{tooltip || "Open version control"}</TooltipContent>
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

function PauseCanvasButton({
  paused,
  onToggle,
  disabled,
  disabledTooltip,
}: {
  paused: boolean;
  onToggle: (paused: boolean) => void;
  disabled?: boolean;
  disabledTooltip?: string;
}) {
  const content = (
    <UIButton
      type="button"
      variant="outline"
      size="sm"
      className={cn("gap-1.5", paused ? "text-amber-600 border-amber-200 bg-amber-50 hover:bg-amber-100" : "")}
      onClick={() => onToggle(!paused)}
      disabled={disabled}
    >
      {paused ? <Play className="h-3.5 w-3.5 fill-current" /> : <Pause className="h-3.5 w-3.5" />}
      {paused ? "Resume Canvas" : "Pause Canvas"}
    </UIButton>
  );

  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{content}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="inline-flex">{content}</div>
      </TooltipTrigger>
      <TooltipContent side="top">
        {paused ? "Resume processing for all triggers" : "Pause all triggers for this canvas"}
      </TooltipContent>
    </Tooltip>
  );
}

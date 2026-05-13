import type { AgentState } from "@/components/AgentSidebar/useAgentState";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { GitBranch, MoreVertical, Pencil, Settings } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../button";
import { AgentSidebarTrigger } from "./components/AgentSidebarTrigger";
import { CanvasModeToggle } from "./components/CanvasModeToggle";
import { EnterEditDraftDropdown } from "./components/EnterEditDraftDropdown";

type HeaderMode = "default" | "launchpad" | "version-live" | "version-edit" | "runs" | "repo";

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
  onSelectRuns?: () => void;
  onSelectLaunchpad?: () => void;
  onSelectRepo?: () => void;
  runsNotificationCount?: number;
  /** Label for the publish/propose-change button in version edit mode. Defaults to "Publish". */
  publishVersionLabel?: string;
  /** When true, shows the Discard control next to Publish in version edit mode (draft differs from live). */
  hasUnpublishedDraftChanges?: boolean;
  /** ISO timestamp of the existing unpublished draft, used to label "Last edited X" in the Edit dropdown. */
  unpublishedDraftUpdatedAt?: string;
  /** Discard the existing draft and start a new edit session from live. Shown in the Edit dropdown when a draft exists. */
  onDiscardDraftAndStartEdit?: () => void;
  /** Canvas settings route requires `canvases:update`; hide the menu when the user cannot update. */
  showCanvasSettingsMenu?: boolean;
  isVersionControlOpen?: boolean;
  onOpenVersionControl?: () => void;
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
  const showCanvasViewModeToggle =
    props.mode === "launchpad" || props.mode === "version-live" || props.mode === "version-edit" || props.mode === "runs" || props.mode === "repo";
  const canvasViewMode = props.mode === "launchpad" ? "launchpad" : props.mode === "runs" ? "runs" : props.mode === "repo" ? "repo" : "version-live";
  const editing = props.mode === "version-edit";

  return (
    <div className="relative flex h-12 items-center border-b border-slate-950/15 bg-slate-100 px-4 gap-3">
      <AgentSidebarTrigger agentState={props.agentState} />

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-16 sm:px-24">
        <div className="pointer-events-auto">
          {showCanvasViewModeToggle && props.onExitEditMode ? (
            <CanvasModeToggle
              mode={canvasViewMode}
              onSelectLaunchpad={props.onSelectLaunchpad}
              onSelectLive={props.onExitEditMode}
              onSelectRuns={props.onSelectRuns}
              onSelectRepo={props.onSelectRepo}
              runsNotificationCount={props.runsNotificationCount}
              editing={editing}
              hasDraft={!!props.hasUnpublishedDraftChanges}
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
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  unpublishedDraftUpdatedAt,
  onDiscardDraftAndStartEdit,
}: HeaderProps) {
  const showVersionControlTrigger = mode === "version-live" && !!onOpenVersionControl;
  const showEditButton = mode === "version-live" && !!onEnterEditMode;
  const showDraftDropdown =
    showEditButton && !!hasUnpublishedDraftChanges && !!onDiscardDraftAndStartEdit && !enterEditModeDisabled;

  return (
    <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
      {showVersionControlTrigger ? (
        <VersionControlButton
          onToggle={onOpenVersionControl}
          isOpen={!!isVersionControlOpen}
          tooltip={versionControlButtonTooltip}
          notificationCount={versionControlNotificationCount}
        />
      ) : null}

      {showEditButton ? (
        showDraftDropdown ? (
          <EnterEditDraftDropdown
            onContinueEditing={onEnterEditMode}
            onDiscardAndStartEdit={onDiscardDraftAndStartEdit!}
            updatedAt={unpublishedDraftUpdatedAt}
          />
        ) : (
          <EnterEditButton
            onClick={onEnterEditMode}
            disabled={!!enterEditModeDisabled}
            disabledTooltip={enterEditModeDisabledTooltip}
          />
        )
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
        <EditModeVersionActions
          hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
          onDiscardVersion={onDiscardVersion}
          discardVersionDisabled={discardVersionDisabled}
          discardVersionDisabledTooltip={discardVersionDisabledTooltip}
          onExitEditMode={onExitEditMode}
          exitEditModeDisabled={exitEditModeDisabled}
          exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
          onPublishVersion={onPublishVersion}
          publishVersionLabel={publishVersionLabel}
          publishVersionDisabled={publishVersionDisabled}
          publishVersionDisabledTooltip={publishVersionDisabledTooltip}
        />
      ) : null}
    </div>
  );
}

function EditModeVersionActions({
  hasUnpublishedDraftChanges,
  onDiscardVersion,
  discardVersionDisabled,
  discardVersionDisabledTooltip,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  onPublishVersion,
  publishVersionLabel,
  publishVersionDisabled,
  publishVersionDisabledTooltip,
}: Pick<
  HeaderProps,
  | "hasUnpublishedDraftChanges"
  | "onDiscardVersion"
  | "discardVersionDisabled"
  | "discardVersionDisabledTooltip"
  | "onExitEditMode"
  | "exitEditModeDisabled"
  | "exitEditModeDisabledTooltip"
  | "onPublishVersion"
  | "publishVersionLabel"
  | "publishVersionDisabled"
  | "publishVersionDisabledTooltip"
>) {
  return (
    <div className="flex items-center gap-2">
      {hasUnpublishedDraftChanges ? (
        <DiscardDraftButton
          onDiscard={() => onDiscardVersion?.()}
          disabled={discardVersionDisabled || !onDiscardVersion}
          disabledTooltip={discardVersionDisabledTooltip}
        />
      ) : null}
      {onExitEditMode ? (
        <ExitEditButton
          onClick={() => onExitEditMode()}
          disabled={!!exitEditModeDisabled}
          disabledTooltip={exitEditModeDisabledTooltip}
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
  );
}

function EnterEditButton({
  onClick,
  disabled,
  disabledTooltip,
}: {
  onClick: () => void;
  disabled: boolean;
  disabledTooltip?: string;
}) {
  const button = (
    <UIButton
      type="button"
      variant="default"
      size="sm"
      onClick={onClick}
      disabled={disabled}
      data-testid="canvas-edit-button"
    >
      <Pencil className="h-3.5 w-3.5" />
      Edit
    </UIButton>
  );

  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{button}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return button;
}

function ExitEditButton({
  onClick,
  disabled,
  disabledTooltip,
}: {
  onClick: () => void;
  disabled: boolean;
  disabledTooltip?: string;
}) {
  const button = (
    <UIButton
      type="button"
      variant="outline"
      size="sm"
      onClick={onClick}
      disabled={disabled}
      data-testid="canvas-exit-edit-button"
    >
      Exit edit
    </UIButton>
  );

  if (disabled && disabledTooltip) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="inline-flex">{button}</div>
        </TooltipTrigger>
        <TooltipContent side="top">{disabledTooltip}</TooltipContent>
      </Tooltip>
    );
  }

  return button;
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

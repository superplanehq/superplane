import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Button as UIButton } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreVertical, Settings } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { CanvasModeToggle } from "./components/CanvasModeToggle";
import { CanvasProjectSwitcher } from "./components/CanvasProjectSwitcher";
import { CanvasToolSidebarTrigger } from "./components/CanvasToolSidebarTrigger";
import { SecondaryHeaderActions, EditModeTopHeaderActions, LiveModeTopHeaderActions } from "./HeaderSecondaryActions";

export type HeaderMode = "default" | "version-live" | "version-edit" | "runs" | "dashboard" | "memory" | "files";

export interface HeaderProps {
  /** Shown centered in the top bar (canvas or template display name). */
  canvasName: string;
  onSave?: () => void;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  onShowDiff?: () => void;
  visualDiffEnabled?: boolean;
  onToggleVisualDiff?: () => void;
  draftVisualDiff?: {
    diffCounts: { added: number; updated: number; removed: number };
    diffToggles: {
      showDeletedNodes: boolean;
      toggleShowDeletedNodes: () => void;
      showEdgeDiff: boolean;
      toggleShowEdgeDiff: () => void;
    };
  };
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
  /** When true, the canvas draft is active regardless of the current Console / Canvas / Memory tab. */
  isEditing?: boolean;
  /** Switches back to the Canvas tab without changing edit mode. */
  onSelectCanvasView?: () => void;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  onSelectDashboard?: () => void;
  /** Provided when Memory is available as a first-class tab; opens the Memory view. */
  onSelectMemory?: () => void;
  /** Provided when Files is available as a first-class tab; opens the Files view. */
  onSelectFiles?: () => void;
  /** When set with `mode === "dashboard"` and editing, shows Add panel in the secondary header. */
  onDashboardAddPanel?: () => void;
  /** When set with `mode === "dashboard"` and editing, shows the YAML button in the secondary header. */
  onDashboardOpenYaml?: () => void;
  /** When set with the Canvas tab active and editing, opens the add-component sidebar. */
  onCanvasAddComponent?: () => void;
  /** When true, the YAML button advertises read-only YAML view. Defaults to editable copy. */
  dashboardYamlReadOnly?: boolean;
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
  toolSidebarState: CanvasToolSidebarState;
}

export function Header(props: HeaderProps) {
  const headerTitle = props.canvasName.trim() || "Canvas";

  return (
    <header>
      <PageHeader
        organizationId={props.organizationId}
        headerTitle={headerTitle}
        showCanvasSettingsMenu={props.showCanvasSettingsMenu}
        mode={props.mode}
        isEditing={props.isEditing}
        hasUnpublishedDraftChanges={props.hasUnpublishedDraftChanges}
        onDiscardVersion={props.onDiscardVersion}
        discardVersionDisabled={props.discardVersionDisabled}
        discardVersionDisabledTooltip={props.discardVersionDisabledTooltip}
        onExitEditMode={props.onExitEditMode}
        exitEditModeDisabled={props.exitEditModeDisabled}
        exitEditModeDisabledTooltip={props.exitEditModeDisabledTooltip}
        onPublishVersion={props.onPublishVersion}
        publishVersionLabel={props.publishVersionLabel}
        publishVersionDisabled={props.publishVersionDisabled}
        publishVersionDisabledTooltip={props.publishVersionDisabledTooltip}
        onEnterEditMode={props.onEnterEditMode}
        enterEditModeDisabled={props.enterEditModeDisabled}
        enterEditModeDisabledTooltip={props.enterEditModeDisabledTooltip}
        onDiscardDraftAndStartEdit={props.onDiscardDraftAndStartEdit}
        unpublishedDraftUpdatedAt={props.unpublishedDraftUpdatedAt}
      />

      <SecondaryHeader {...props} />
    </header>
  );
}

function PageHeader({
  organizationId,
  headerTitle,
  showCanvasSettingsMenu = true,
  mode,
  isEditing = false,
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
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onDiscardDraftAndStartEdit,
  unpublishedDraftUpdatedAt,
}: {
  organizationId?: string;
  headerTitle: string;
  showCanvasSettingsMenu?: boolean;
  mode?: HeaderMode;
  isEditing?: boolean;
  hasUnpublishedDraftChanges?: boolean;
  onDiscardVersion?: () => void;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  onPublishVersion?: () => void;
  publishVersionLabel?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onDiscardDraftAndStartEdit?: () => void;
  unpublishedDraftUpdatedAt?: string;
}) {
  const navigate = useNavigate();
  const { workflowId, canvasId: canvasIdParam } = useParams<{ workflowId?: string; canvasId?: string }>();
  const activeCanvasId = canvasIdParam || workflowId;

  return (
    <div className="relative z-20 flex h-10 items-center border-b border-slate-950/15 px-3 sm:px-4">
      <div className="relative z-10 flex min-w-0 shrink-0 items-center">
        <OrganizationMenuButton organizationId={organizationId} />
      </div>
      <div className="pointer-events-none absolute inset-x-0 flex items-center justify-center px-24">
        <div className="pointer-events-auto">
          {organizationId && activeCanvasId ? (
            <CanvasProjectSwitcher
              organizationId={organizationId}
              activeCanvasId={activeCanvasId}
              canvasName={headerTitle}
            />
          ) : (
            <span className="block truncate text-center text-[13px] font-medium text-slate-900">{headerTitle}</span>
          )}
        </div>
      </div>
      <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
        {mode !== "runs" && !isEditing && onEnterEditMode ? (
          <LiveModeTopHeaderActions
            onEnterEditMode={onEnterEditMode}
            enterEditModeDisabled={enterEditModeDisabled}
            enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
            hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
            onDiscardDraftAndStartEdit={onDiscardDraftAndStartEdit}
            unpublishedDraftUpdatedAt={unpublishedDraftUpdatedAt}
          />
        ) : null}
        {isEditing ? (
          <EditModeTopHeaderActions
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
    (!!props.onSelectDashboard || !!props.onSelectMemory || !!props.onSelectFiles) &&
    (props.mode === "version-live" ||
      props.mode === "runs" ||
      props.mode === "dashboard" ||
      props.mode === "memory" ||
      props.mode === "files");
  const canvasViewMode =
    props.mode === "runs"
      ? "runs"
      : props.mode === "dashboard"
        ? "dashboard"
        : props.mode === "memory"
          ? "memory"
          : props.mode === "files"
            ? "files"
            : "version-live";
  const editing = props.isEditing ?? props.mode === "version-edit";

  return (
    <div className="relative z-10 flex h-10 items-center gap-3 border-b border-slate-950/15 bg-white px-4">
      <CanvasToolSidebarTrigger toolSidebarState={props.toolSidebarState} />

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-16 sm:px-24">
        <div className="pointer-events-auto">
          {showCanvasViewModeToggle && props.onSelectCanvasView ? (
            <CanvasModeToggle
              mode={canvasViewMode}
              onSelectLive={props.onSelectCanvasView}
              onSelectDashboard={props.onSelectDashboard}
              onSelectMemory={props.onSelectMemory}
              onSelectFiles={props.onSelectFiles}
              editing={editing}
            />
          ) : null}
        </div>
      </div>

      <SecondaryHeaderActions {...props} />
    </div>
  );
}

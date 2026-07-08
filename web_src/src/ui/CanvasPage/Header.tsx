import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import type { CanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { useParams } from "react-router-dom";
import { CanvasModeToggle, type CanvasMode } from "./components/CanvasModeToggle";
import { CanvasProjectSwitcher } from "./components/CanvasProjectSwitcher";
import { CanvasRunsSidebarTrigger } from "./components/CanvasRunsSidebarTrigger";
import { CanvasVersionsSidebarTrigger } from "./components/CanvasVersionsSidebarTrigger";
import { CanvasToolSidebarTrigger } from "./components/CanvasToolSidebarTrigger";
import { SecondaryHeaderActions, EditModeTopHeaderActions, LiveModeTopHeaderActions } from "./HeaderSecondaryActions";

export type HeaderMode = "default" | "version-live" | "console" | "memory" | "files";

export interface HeaderProps {
  /** Shown centered in the top bar (canvas or template display name). May be undefined while the canvas is still loading. */
  canvasName?: string;
  onPublishVersion?: () => void;
  onDiscardVersion?: () => void;
  onShowDiff?: () => void;
  onShowConsoleDiff?: () => void;
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
  draftConsoleDiff?: {
    diffCounts: { added: number; updated: number; removed: number };
  };
  organizationId?: string;
  publishVersionDisabled?: boolean;
  publishVersionDisabledTooltip?: string;
  discardVersionDisabled?: boolean;
  discardVersionDisabledTooltip?: string;
  /** True when the active edit session has uncommitted staged spec edits (shows Commit/Reset). */
  hasStagingChanges?: boolean;
  /** Staging is based on an outdated main-branch commit; only discard is allowed. */
  stagingStale?: boolean;
  /** Opens the commit dialog for staged edits. */
  onCommitStaging?: () => void;
  commitStagingPending?: boolean;
  resetStagingPending?: boolean;
  /** Discard staged edits, reverting to the last commit. */
  onResetStaging?: () => void;
  /** Discard stale staging after main moved forward. */
  onDiscardStaleStaging?: () => void;
  discardStaleStagingPending?: boolean;
  mode?: HeaderMode;
  /** When true, the canvas draft is active regardless of the current Console / Canvas / Memory tab. */
  isEditing?: boolean;
  /** True while an edit session is active (editing a draft or previewing a version from the versions sidebar). Controls the Edit/Exit affordance. */
  isEditSessionActive?: boolean;
  /** Switches back to the Canvas tab without changing edit mode. */
  onSelectCanvasView?: () => void;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  onSelectConsole?: () => void;
  /** Provided when Memory is available as a first-class tab; opens the Memory view. */
  onSelectMemory?: () => void;
  /** Provided when Files is available as a first-class tab; opens the Files view. */
  onSelectFiles?: () => void;
  /** DOM slot for Files mode actions owned by the files editor overlay. */
  filesHeaderActionsSlotId?: string;
  /** Label for the publish/propose-change button in version edit mode. Defaults to "Publish". */
  publishVersionLabel?: string;
  /** When true, shows the Discard control next to Publish in version edit mode (draft differs from live). */
  hasUnpublishedDraftChanges?: boolean;
  /** Draft indicator for the Canvas tab when workflow graph changes exist. */
  hasUnpublishedCanvasDraftChanges?: boolean;
  /** Draft indicator for the Console tab when console changes exist. */
  hasUnpublishedConsoleDraftChanges?: boolean;
  /** Draft indicator for the Files tab when a non-spec repository file is staged. */
  hasFilesStagingChanges?: boolean;
  hasUncommittedCanvasDraftChanges?: boolean;
  hasUncommittedConsoleDraftChanges?: boolean;
  hasUncommittedFilesDraftChanges?: boolean;
  hasCommittedCanvasDraftChanges?: boolean;
  hasCommittedConsoleDraftChanges?: boolean;
  hasCommittedFilesDraftChanges?: boolean;
  editTabTone?: "uncommitted" | "ready" | "neutral";
  activeDraftBranchLabel?: string;
  activeDraftBranchShortSha?: string;
  /** Canvas rename requires `canvases:update`; hide rename when the user cannot update. */
  showCanvasSettingsMenu?: boolean;
  toolSidebarState: CanvasToolSidebarState;
  runsSidebarState: CanvasRunsSidebarState;
  versionsSidebarState: CanvasVersionsSidebarState;
}

export function Header(props: HeaderProps) {
  const headerTitle = (props.canvasName ?? "").trim() || "Canvas";

  return (
    <header>
      <PageHeader
        organizationId={props.organizationId}
        headerTitle={headerTitle}
        isEditing={props.isEditing}
        isEditSessionActive={props.isEditSessionActive}
        onExitEditMode={props.onExitEditMode}
        exitEditModeDisabled={props.exitEditModeDisabled}
        exitEditModeDisabledTooltip={props.exitEditModeDisabledTooltip}
        onEnterEditMode={props.onEnterEditMode}
        enterEditModeDisabled={props.enterEditModeDisabled}
        enterEditModeDisabledTooltip={props.enterEditModeDisabledTooltip}
        activeDraftBranchLabel={props.activeDraftBranchLabel}
        activeDraftBranchShortSha={props.activeDraftBranchShortSha}
        showCanvasSettingsMenu={props.showCanvasSettingsMenu}
      />

      <SecondaryHeader {...props} />
    </header>
  );
}

interface PageHeaderBarProps {
  organizationId?: string;
  headerTitle: string;
  showCanvasSettingsMenu?: boolean;
  isEditing?: boolean;
  isEditSessionActive?: boolean;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  activeDraftBranchLabel?: string;
  activeDraftBranchShortSha?: string;
}

function PageHeader({
  organizationId,
  headerTitle,
  isEditing = false,
  isEditSessionActive,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  activeDraftBranchLabel,
  activeDraftBranchShortSha,
  showCanvasSettingsMenu = true,
}: PageHeaderBarProps) {
  const {
    workflowId,
    canvasId: canvasIdParam,
    appId,
  } = useParams<{
    workflowId?: string;
    canvasId?: string;
    appId?: string;
  }>();
  const activeCanvasId = appId || canvasIdParam || workflowId;
  const inEditSession = isEditSessionActive ?? isEditing;

  return (
    <div className="relative z-20 flex h-10 items-center border-b border-slate-950/15 px-2 sm:px-3 dark:border-gray-700/70">
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
              canUpdateCanvas={showCanvasSettingsMenu}
            />
          ) : (
            <span className="block truncate text-center text-[13px] font-medium text-slate-900 dark:text-gray-100">
              {headerTitle}
            </span>
          )}
        </div>
      </div>
      <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
        {inEditSession ? (
          <div className="flex items-center">
            {activeDraftBranchLabel ? (
              <span
                className="hidden text-[13px] font-medium text-slate-600 sm:inline dark:text-gray-400"
                data-testid="active-draft-branch-chip"
              >
                Editing: {activeDraftBranchLabel}
                {activeDraftBranchShortSha ? ` @ ${activeDraftBranchShortSha}` : ""}
              </span>
            ) : null}
            <EditModeTopHeaderActions
              onExitEditMode={onExitEditMode}
              exitEditModeDisabled={exitEditModeDisabled}
              exitEditModeDisabledTooltip={exitEditModeDisabledTooltip}
            />
          </div>
        ) : null}
        {!inEditSession && onEnterEditMode ? (
          <LiveModeTopHeaderActions
            onEnterEditMode={onEnterEditMode}
            enterEditModeDisabled={enterEditModeDisabled}
            enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
          />
        ) : null}
      </div>
    </div>
  );
}

function SecondaryHeader(props: HeaderProps) {
  const showCanvasViewModeToggle = shouldShowCanvasViewModeToggle(props);
  const canvasViewMode = getCanvasViewMode(props.mode);
  const editing = props.isEditing ?? false;
  const editTabTone: HeaderProps["editTabTone"] = editing ? "uncommitted" : (props.editTabTone ?? "neutral");

  return (
    <div className="relative z-10 flex h-10 items-center gap-3 border-b border-slate-950/15 bg-white px-3 dark:border-gray-700/70 dark:bg-gray-900">
      <div className="relative z-10 -ml-1.5 flex h-7 shrink-0 items-center gap-1">
        <CanvasToolSidebarTrigger toolSidebarState={props.toolSidebarState} />
        <CanvasRunsSidebarTrigger runsSidebarState={props.runsSidebarState} />
        <CanvasVersionsSidebarTrigger versionsSidebarState={props.versionsSidebarState} />
      </div>

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-16 sm:px-24">
        <div className="pointer-events-auto">
          {showCanvasViewModeToggle && props.onSelectCanvasView ? (
            <CanvasModeToggle
              mode={canvasViewMode}
              onSelectLive={props.onSelectCanvasView}
              onSelectConsole={props.onSelectConsole}
              onSelectMemory={props.onSelectMemory}
              onSelectFiles={props.onSelectFiles}
              editing={editing}
              hasCanvasUncommitted={!!props.hasUncommittedCanvasDraftChanges}
              hasCanvasCommitted={!!props.hasCommittedCanvasDraftChanges}
              hasConsoleUncommitted={!!props.hasUncommittedConsoleDraftChanges}
              hasConsoleCommitted={!!props.hasCommittedConsoleDraftChanges}
              hasFilesUncommitted={!!props.hasUncommittedFilesDraftChanges}
              hasFilesCommitted={!!props.hasCommittedFilesDraftChanges}
              editTabTone={editTabTone}
            />
          ) : null}
        </div>
      </div>

      <SecondaryHeaderActions {...props} />
    </div>
  );
}

function shouldShowCanvasViewModeToggle(props: HeaderProps): boolean {
  if (!props.onSelectConsole && !props.onSelectMemory && !props.onSelectFiles) {
    return false;
  }

  return isCanvasViewMode(props.mode);
}

function isCanvasViewMode(mode: HeaderMode | undefined): boolean {
  return (
    !mode ||
    mode === "default" ||
    mode === "version-live" ||
    mode === "console" ||
    mode === "memory" ||
    mode === "files"
  );
}

function getCanvasViewMode(mode: HeaderMode | undefined): CanvasMode {
  if (mode === "console" || mode === "memory" || mode === "files") {
    return mode;
  }

  return "version-live";
}

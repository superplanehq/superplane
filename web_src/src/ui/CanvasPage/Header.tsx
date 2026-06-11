import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import type { CanvasesCanvasVersion } from "@/api-client";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { useParams } from "react-router-dom";
import { CanvasModeToggle, type CanvasMode } from "./components/CanvasModeToggle";
import { CanvasProjectSwitcher } from "./components/CanvasProjectSwitcher";
import { CanvasToolSidebarTrigger } from "./components/CanvasToolSidebarTrigger";
import { SecondaryHeaderActions, EditModeTopHeaderActions, LiveModeTopHeaderActions } from "./HeaderSecondaryActions";

export type HeaderMode =
  | "default"
  | "version-live"
  | "version-edit"
  | "runs"
  | "versions"
  | "console"
  | "memory"
  | "files";

export interface HeaderProps {
  /** Shown centered in the top bar (canvas or template display name). May be undefined while the canvas is still loading. */
  canvasName?: string;
  onSave?: () => void;
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
  onSelectConsole?: () => void;
  /** Provided when Versions is available as a first-class tab; opens the Versions view. */
  onSelectVersions?: () => void;
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
  /** ISO timestamp of the existing unpublished draft, used to label "Last edited X" in the Edit dropdown. */
  unpublishedDraftUpdatedAt?: string;
  /** Discard the existing draft and start a new edit session from live. Shown in the Edit dropdown when a draft exists. */
  onDiscardDraftAndStartEdit?: () => void;
  startEditingDrafts?: CanvasesCanvasVersion[];
  startEditingDefaultDraft?: CanvasesCanvasVersion | null;
  startEditingMenuOpen?: boolean;
  onStartEditingMenuOpenChange?: (open: boolean) => void;
  onContinueDraftBranch?: (branchName: string) => void;
  onCreateDraftBranch?: () => void;
  createDraftBranchPending?: boolean;
  activeDraftBranchLabel?: string;
  activeDraftBranchShortSha?: string;
  /** Canvas rename requires `canvases:update`; hide rename when the user cannot update. */
  showCanvasSettingsMenu?: boolean;
  toolSidebarState: CanvasToolSidebarState;
}

export function Header(props: HeaderProps) {
  const headerTitle = (props.canvasName ?? "").trim() || "Canvas";

  return (
    <header>
      <PageHeader
        organizationId={props.organizationId}
        headerTitle={headerTitle}
        mode={props.mode}
        isEditing={props.isEditing}
        hasUnpublishedDraftChanges={props.hasUnpublishedDraftChanges}
        onExitEditMode={props.onExitEditMode}
        exitEditModeDisabled={props.exitEditModeDisabled}
        exitEditModeDisabledTooltip={props.exitEditModeDisabledTooltip}
        onEnterEditMode={props.onEnterEditMode}
        enterEditModeDisabled={props.enterEditModeDisabled}
        enterEditModeDisabledTooltip={props.enterEditModeDisabledTooltip}
        onDiscardDraftAndStartEdit={props.onDiscardDraftAndStartEdit}
        unpublishedDraftUpdatedAt={props.unpublishedDraftUpdatedAt}
        startEditingDrafts={props.startEditingDrafts}
        startEditingDefaultDraft={props.startEditingDefaultDraft}
        startEditingMenuOpen={props.startEditingMenuOpen}
        onStartEditingMenuOpenChange={props.onStartEditingMenuOpenChange}
        onContinueDraftBranch={props.onContinueDraftBranch}
        onCreateDraftBranch={props.onCreateDraftBranch}
        createDraftBranchPending={props.createDraftBranchPending}
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
  mode?: HeaderMode;
  isEditing?: boolean;
  hasUnpublishedDraftChanges?: boolean;
  onExitEditMode?: () => void;
  exitEditModeDisabled?: boolean;
  exitEditModeDisabledTooltip?: string;
  onEnterEditMode?: () => void;
  enterEditModeDisabled?: boolean;
  enterEditModeDisabledTooltip?: string;
  onDiscardDraftAndStartEdit?: () => void;
  unpublishedDraftUpdatedAt?: string;
  startEditingDrafts?: CanvasesCanvasVersion[];
  startEditingDefaultDraft?: CanvasesCanvasVersion | null;
  startEditingMenuOpen?: boolean;
  onStartEditingMenuOpenChange?: (open: boolean) => void;
  onContinueDraftBranch?: (branchName: string) => void;
  onCreateDraftBranch?: () => void;
  createDraftBranchPending?: boolean;
  activeDraftBranchLabel?: string;
  activeDraftBranchShortSha?: string;
}

function PageHeader({
  organizationId,
  headerTitle,
  isEditing = false,
  hasUnpublishedDraftChanges,
  onExitEditMode,
  exitEditModeDisabled,
  exitEditModeDisabledTooltip,
  onEnterEditMode,
  enterEditModeDisabled,
  enterEditModeDisabledTooltip,
  onDiscardDraftAndStartEdit,
  unpublishedDraftUpdatedAt,
  startEditingDrafts,
  startEditingDefaultDraft,
  startEditingMenuOpen,
  onStartEditingMenuOpenChange,
  onContinueDraftBranch,
  onCreateDraftBranch,
  createDraftBranchPending,
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

  return (
    <div className="relative z-20 flex h-10 items-center border-b border-slate-950/15 px-2 sm:px-3">
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
            <span className="block truncate text-center text-[13px] font-medium text-slate-900">{headerTitle}</span>
          )}
        </div>
      </div>
      <div className="relative z-10 ml-auto flex shrink-0 items-center gap-2">
        {isEditing ? (
          <div className="flex items-center">
            {activeDraftBranchLabel ? (
              <span
                className="hidden text-[13px] font-medium text-slate-600 sm:inline"
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
        {!isEditing && (onEnterEditMode || startEditingDrafts !== undefined) ? (
          <LiveModeTopHeaderActions
            onEnterEditMode={onEnterEditMode}
            enterEditModeDisabled={enterEditModeDisabled}
            enterEditModeDisabledTooltip={enterEditModeDisabledTooltip}
            hasUnpublishedDraftChanges={hasUnpublishedDraftChanges}
            onDiscardDraftAndStartEdit={onDiscardDraftAndStartEdit}
            unpublishedDraftUpdatedAt={unpublishedDraftUpdatedAt}
            startEditingDrafts={startEditingDrafts}
            startEditingDefaultDraft={startEditingDefaultDraft}
            startEditingMenuOpen={startEditingMenuOpen}
            onStartEditingMenuOpenChange={onStartEditingMenuOpenChange}
            onContinueDraftBranch={onContinueDraftBranch}
            onCreateDraftBranch={onCreateDraftBranch}
            createDraftBranchPending={createDraftBranchPending}
          />
        ) : null}
      </div>
    </div>
  );
}

function SecondaryHeader(props: HeaderProps) {
  const showCanvasViewModeToggle = shouldShowCanvasViewModeToggle(props);
  const canvasViewMode = getCanvasViewMode(props.mode);
  const editing = props.isEditing ?? props.mode === "version-edit";

  return (
    <div className="relative z-10 flex h-10 items-center gap-3 border-b border-slate-950/15 bg-white px-3">
      <CanvasToolSidebarTrigger toolSidebarState={props.toolSidebarState} />

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-16 sm:px-24">
        <div className="pointer-events-auto">
          {showCanvasViewModeToggle && props.onSelectCanvasView ? (
            <CanvasModeToggle
              mode={canvasViewMode}
              onSelectLive={props.onSelectCanvasView}
              onSelectVersions={props.onSelectVersions}
              onSelectConsole={props.onSelectConsole}
              onSelectMemory={props.onSelectMemory}
              onSelectFiles={props.onSelectFiles}
              editing={editing}
              hasDraft={props.hasUnpublishedCanvasDraftChanges ?? !!props.hasUnpublishedDraftChanges}
              hasConsoleDraft={!!props.hasUnpublishedConsoleDraftChanges}
            />
          ) : null}
        </div>
      </div>

      <SecondaryHeaderActions {...props} />
    </div>
  );
}

function shouldShowCanvasViewModeToggle(props: HeaderProps): boolean {
  if (!props.onSelectConsole && !props.onSelectVersions && !props.onSelectMemory && !props.onSelectFiles) {
    return false;
  }

  return isCanvasViewMode(props.mode);
}

function isCanvasViewMode(mode: HeaderMode | undefined): boolean {
  return (
    mode === "version-live" ||
    mode === "version-edit" ||
    mode === "versions" ||
    mode === "console" ||
    mode === "memory" ||
    mode === "files"
  );
}

function getCanvasViewMode(mode: HeaderMode | undefined): CanvasMode {
  if (mode === "versions" || mode === "console" || mode === "memory" || mode === "files") {
    return mode;
  }

  return "version-live";
}

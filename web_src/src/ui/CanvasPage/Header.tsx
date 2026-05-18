import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { Button as UIButton } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { MoreVertical, Settings } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { CanvasModeToggle } from "./components/CanvasModeToggle";
import { CanvasToolSidebarTrigger } from "./components/CanvasToolSidebarTrigger";
import { SecondaryHeaderActions } from "./HeaderSecondaryActions";

export type HeaderMode = "default" | "version-live" | "version-edit" | "runs" | "dashboard";

export interface HeaderProps {
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
  onSelectDashboard?: () => void;
  /** When set with `mode === "dashboard"`, shows Add panel in the secondary header. */
  onDashboardAddPanel?: () => void;
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
    !!props.onSelectDashboard &&
    (props.mode === "version-live" ||
      props.mode === "version-edit" ||
      props.mode === "runs" ||
      props.mode === "dashboard");
  const canvasViewMode = props.mode === "runs" ? "runs" : props.mode === "dashboard" ? "dashboard" : "version-live";
  const editing = props.mode === "version-edit";

  return (
    <div className="relative flex h-12 items-center gap-3 border-b border-slate-950/15 bg-slate-100 px-4">
      <CanvasToolSidebarTrigger toolSidebarState={props.toolSidebarState} />

      <div className="pointer-events-none absolute inset-x-0 flex justify-center px-16 sm:px-24">
        <div className="pointer-events-auto">
          {showCanvasViewModeToggle && props.onExitEditMode ? (
            <CanvasModeToggle
              mode={canvasViewMode}
              onSelectLive={props.onExitEditMode}
              onSelectDashboard={props.onSelectDashboard}
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

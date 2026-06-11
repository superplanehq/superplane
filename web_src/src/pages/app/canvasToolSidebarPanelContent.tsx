import type {
  CanvasChangeManagement,
  CanvasesCanvasChangeRequest,
  CanvasesCanvasVersion,
  CanvasesCanvasRun,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { RunsTabPanel } from "@/components/CanvasToolSidebar/RunsTabPanel";
import { VersionsTabPanel } from "@/components/CanvasToolSidebar/VersionsTabPanel";
import type { CanvasVersionNodeDiffContext } from "@/pages/app/CanvasVersionNodeDiffDialog";
import type { ReactNode } from "react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";

export interface CanvasRunsSidebarPanelConfig {
  isOpen: boolean;
  canvasId: string;
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  onSelectRun: (runId: string) => void;
  onNavigateRun?: (runId: string) => void;
  onSelectLiveCanvas: () => void;
  onBackToRunList?: () => void;
  initialOpenDetail?: boolean;
  detailDismissedForRunId?: string | null;
  selectedNodeId?: string | null;
  onSelectNode?: (nodeId: string) => void;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
  workflowNodes?: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  onStatusFiltersChange?: (filters: RunStatusFilter[]) => void;
}

export interface CanvasVersionsSidebarPanelConfig {
  isOpen: boolean;
  scrollPersistenceKey?: string;
  liveCanvasVersionId?: string;
  liveCanvasVersion?: CanvasesCanvasVersion | null;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  pendingApprovalVersions?: Array<{
    version: CanvasesCanvasVersion;
    changeRequest: CanvasesCanvasChangeRequest;
  }>;
  rejectedVersions?: Array<{
    version: CanvasesCanvasVersion;
    changeRequest: CanvasesCanvasChangeRequest;
  }>;
  liveVersions: CanvasesCanvasVersion[];
  liveVersionChangeRequestsByVersionId?: Map<string, CanvasesCanvasChangeRequest>;
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onVersionNodeDiffContextChange: (context: CanvasVersionNodeDiffContext | null) => void;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
  changeRequestApprovalConfig?: CanvasChangeManagement;
  draftBranches?: CanvasesCanvasVersion[];
  activeDraftBranch?: string | null;
  onOpenDraftBranch?: (branchName: string) => void;
  onDeleteDraftBranch?: (versionId: string) => void;
  deleteDraftBranchPending?: boolean;
}

export function renderCanvasRunsSidebarPanel(config: CanvasRunsSidebarPanelConfig): ReactNode {
  if (!config.isOpen) return null;
  const { isOpen: _isOpen, ...props } = config;
  return <RunsTabPanel {...props} />;
}

export function renderCanvasVersionsSidebarPanel(config: CanvasVersionsSidebarPanelConfig): ReactNode {
  if (!config.isOpen) return null;
  const { isOpen: _isOpen, ...props } = config;
  return <VersionsTabPanel {...props} />;
}

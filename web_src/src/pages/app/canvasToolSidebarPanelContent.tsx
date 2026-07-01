import type {
  CanvasesCanvasBranch,
  CanvasesCanvasRun,
  CanvasesCanvasVersion,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { RunsTabPanel } from "@/components/CanvasToolSidebar/RunsTabPanel";
import { VersionsTabPanel } from "@/components/CanvasToolSidebar/VersionsTabPanel";
import type { ReactNode } from "react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";

export interface CanvasRunsSidebarPanelConfig {
  isOpen: boolean;
  canvasId: string;
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  selectedRun?: CanvasesCanvasRun | null;
  isSelectedRunLoading?: boolean;
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
  organizationId?: string;
  isOpen: boolean;
  scrollPersistenceKey?: string;
  branchHeadVersionId?: string;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  branchCommits: CanvasesCanvasVersion[];
  canUpdateCanvas: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onLoadMoreBranchCommits?: () => void;
  loadMoreBranchCommitsDisabled?: boolean;
  loadMoreBranchCommitsPending?: boolean;
  canvasBranches?: CanvasesCanvasBranch[];
  activeBranchName?: string;
  onSelectBranch?: (branchName: string) => void;
  branchSelectorDisabled?: boolean;
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

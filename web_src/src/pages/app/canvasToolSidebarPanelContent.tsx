import type { CanvasesCanvasVersion, CanvasesCanvasRun } from "@/api-client";
import { RunsTabPanelContent } from "@/components/CanvasToolSidebar/RunsTabPanel";
import type { RunFiltersState } from "@/components/CanvasToolSidebar/useRunFilters";
import { VersionsTabPanel } from "@/components/CanvasToolSidebar/VersionsTabPanel";
import type { ReactNode } from "react";

export interface CanvasRunsSidebarPanelConfig {
  isOpen: boolean;
  canvasId: string;
  runs: CanvasesCanvasRun[];
  selectedRunId: string | null;
  selectedRun?: CanvasesCanvasRun | null;
  isSelectedRunLoading?: boolean;
  onSelectRun: (runId: string) => void;
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
  componentIconMap?: Record<string, string>;
  filterState: RunFiltersState;
}

export interface CanvasVersionsSidebarPanelConfig {
  isOpen: boolean;
  scrollPersistenceKey?: string;
  liveCanvasVersionId?: string;
  liveCanvasVersion?: CanvasesCanvasVersion | null;
  selectedCanvasVersion?: CanvasesCanvasVersion | null;
  liveVersions: CanvasesCanvasVersion[];
  canEditCanvasVersion: boolean;
  canvasDeletedRemotely: boolean;
  onUseVersion: (versionID: string) => void;
  onLoadMoreLiveVersions?: () => void;
  loadMoreLiveVersionsDisabled?: boolean;
  loadMoreLiveVersionsPending?: boolean;
}

export function renderCanvasRunsSidebarPanel(config: CanvasRunsSidebarPanelConfig): ReactNode {
  if (!config.isOpen) return null;
  const { isOpen: _isOpen, ...props } = config;
  return <RunsTabPanelContent {...props} />;
}

export function renderCanvasVersionsSidebarPanel(config: CanvasVersionsSidebarPanelConfig): ReactNode {
  if (!config.isOpen) return null;
  const { isOpen: _isOpen, canEditCanvasVersion, ...props } = config;
  return <VersionsTabPanel {...props} canEditCanvasVersion={canEditCanvasVersion} />;
}

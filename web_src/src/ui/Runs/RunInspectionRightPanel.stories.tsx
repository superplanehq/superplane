import type { Meta, StoryObj } from "@storybook/react-vite";
import { useCallback, useMemo, useRef, useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { toast } from "sonner";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { LiveCanvasSidebarRow } from "@/components/CanvasToolSidebar/LiveCanvasSidebarRow";
import { RunsTabListView } from "@/components/CanvasToolSidebar/RunsTabListView";
import { useRunFilters } from "@/components/CanvasToolSidebar/useRunFilters";
import { cn } from "@/lib/utils";
import { RunPanel, type RunDetailContext, type RunDisplayMode } from "./RunPanel";
import { shortId } from "./runPresentation";
import { RunCanvas } from "./storybooks/RunCanvas";
import {
  DEPLOY_NODE_ID,
  RUNS_STORY_CANVAS_ID,
  RunsStorySeed,
  mockRuns,
  mockWorkflowNodes,
} from "./storybooks/fixtures";

/**
 * Runs list stays permanently in the left sidebar. Selecting a run opens a
 * run-detail "page" from the right that reads like a dedicated page: a
 * peek-style chrome row (display-mode toggle + prev/next + overflow), a
 * run-focused identity header, an at-a-glance summary strip, a conditional
 * failure banner, and the filterable run-step accordion. Toggling the
 * display mode expands the panel to a full-width page (canvas hidden).
 */

function RunsListSidebar({
  runs,
  workflowNodes,
  selectedRunId,
  onSelectRun,
  onSelectLiveCanvas,
}: {
  runs: CanvasesCanvasRun[];
  workflowNodes: ComponentsNode[];
  selectedRunId: string | null;
  onSelectRun: (runId: string) => void;
  onSelectLiveCanvas: () => void;
}) {
  const filters = useRunFilters({ runs, workflowNodes, componentIconMap: {} });
  const scrollRef = useRef<HTMLDivElement>(null);

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <LiveCanvasSidebarRow isSelected={!selectedRunId} onSelect={onSelectLiveCanvas} />
      <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
        <RunsTabListView
          isActive
          scrollRef={scrollRef}
          onScroll={() => {}}
          runs={runs}
          filteredRuns={filters.filteredRuns}
          orderedRuns={filters.orderedRuns}
          selectedRunId={selectedRunId}
          onSelectRun={onSelectRun}
          componentIconMap={{}}
          onClearFilters={filters.clearFilters}
          hasAnyFilter={filters.hasAnyFilter}
          selectedStatuses={filters.selectedStatuses}
          selectedTriggerIds={filters.selectedTriggerIds}
          triggerOptions={filters.triggerOptions}
          searchQuery={filters.searchQuery}
          onSearchChange={filters.setSearchQuery}
          onToggleStatus={filters.toggleStatus}
          onClearStatuses={filters.clearStatuses}
          onToggleTrigger={filters.toggleTrigger}
          onClearTriggers={filters.clearTriggers}
        />
      </div>
    </div>
  );
}

function CanvasPlaceholder() {
  return (
    <div className="flex min-h-0 flex-1 items-center justify-center bg-slate-50">
      <div className="max-w-sm text-center text-sm text-slate-400">Select a run from the sidebar to inspect it.</div>
    </div>
  );
}

function RunInspectionRightPanelPlayground({
  initialRunId = null,
  initialNodeId = null,
  initialDisplayMode = "split",
  context = "inspection",
}: {
  initialRunId?: string | null;
  initialNodeId?: string | null;
  initialDisplayMode?: RunDisplayMode;
  context?: RunDetailContext;
}) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(initialRunId);
  const [expandedNodeId, setExpandedNodeId] = useState<string | null>(initialNodeId);
  const [displayMode, setDisplayMode] = useState<RunDisplayMode>(initialDisplayMode);

  const selectedRun = useMemo(() => mockRuns.find((run) => run.id === selectedRunId) ?? null, [selectedRunId]);
  const selectedIndex = useMemo(() => mockRuns.findIndex((run) => run.id === selectedRunId), [selectedRunId]);

  const handleSelectRun = useCallback((runId: string) => {
    setSelectedRunId(runId);
    setExpandedNodeId(null);
  }, []);

  const handleSelectLiveCanvas = useCallback(() => {
    setSelectedRunId(null);
    setExpandedNodeId(null);
    setDisplayMode("split");
  }, []);

  const toggleNode = useCallback((nodeId: string) => {
    setExpandedNodeId((current) => (current === nodeId ? null : nodeId));
  }, []);

  const expandNode = useCallback((nodeId: string) => setExpandedNodeId(nodeId), []);

  const goToRunAt = useCallback((index: number) => {
    const run = mockRuns[index];
    if (!run?.id) return;
    setSelectedRunId(run.id);
    setExpandedNodeId(null);
  }, []);

  const isFull = displayMode === "full" && Boolean(selectedRun);
  const panelWidthClass = displayMode === "full" ? "w-full" : displayMode === "min" ? "w-1/3" : "w-1/2";

  return (
    <div className="flex h-screen min-h-0 bg-white">
      {!isFull ? (
        <CanvasRunsSidebar isOpen>
          <RunsListSidebar
            runs={mockRuns}
            workflowNodes={mockWorkflowNodes}
            selectedRunId={selectedRunId}
            onSelectRun={handleSelectRun}
            onSelectLiveCanvas={handleSelectLiveCanvas}
          />
        </CanvasRunsSidebar>
      ) : null}

      {!isFull ? (
        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          {selectedRun ? (
            <RunCanvas run={selectedRun} selectedNodeId={expandedNodeId} onSelectNode={expandNode} />
          ) : (
            <CanvasPlaceholder />
          )}
        </div>
      ) : null}

      {selectedRun ? (
        <aside className={cn("flex h-full min-w-0 shrink-0 flex-col border-l border-border bg-white", panelWidthClass)}>
          <RunPanel
            canvasId={RUNS_STORY_CANVAS_ID}
            run={selectedRun}
            workflowNodes={mockWorkflowNodes}
            context={context}
            expandedNodeId={expandedNodeId}
            onToggleNode={toggleNode}
            onExpandNode={expandNode}
            onClose={handleSelectLiveCanvas}
            displayMode={displayMode}
            onSetDisplayMode={setDisplayMode}
            onPrevRun={() => goToRunAt(selectedIndex - 1)}
            onNextRun={() => goToRunAt(selectedIndex + 1)}
            hasPrevRun={selectedIndex > 0}
            hasNextRun={selectedIndex >= 0 && selectedIndex < mockRuns.length - 1}
            onViewOnCanvas={() =>
              toast.info("Inspect on canvas", {
                description: `Mock: highlight run ${shortId(selectedRun.id)} on the canvas`,
              })
            }
            onAskAgent={() =>
              toast.info("Ask agent", {
                description: `Mock: open agent with run ${shortId(selectedRun.id)} mentioned`,
              })
            }
          />
        </aside>
      ) : null}
    </div>
  );
}

const meta = {
  title: "Runs Proto/Run Inspection (Right Panel)",
  component: RunInspectionRightPanelPlayground,
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <RunsStorySeed>
        <MemoryRouter>
          <Story />
        </MemoryRouter>
      </RunsStorySeed>
    ),
  ],
} satisfies Meta<typeof RunInspectionRightPanelPlayground>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => <RunInspectionRightPanelPlayground />,
};

export const RunSelected: Story = {
  render: () => <RunInspectionRightPanelPlayground initialRunId="run-passed" />,
};

export const StepExpanded: Story = {
  render: () => <RunInspectionRightPanelPlayground initialRunId="run-passed" initialNodeId={DEPLOY_NODE_ID} />,
};

export const FailedRun: Story = {
  render: () => <RunInspectionRightPanelPlayground initialRunId="run-failed" />,
};

export const RunningRun: Story = {
  render: () => <RunInspectionRightPanelPlayground initialRunId="run-running" />,
};

export const FullPage: Story = {
  render: () => <RunInspectionRightPanelPlayground initialRunId="run-failed" initialDisplayMode="full" />,
};

export const Minimized: Story = {
  render: () => <RunInspectionRightPanelPlayground initialRunId="run-passed" initialDisplayMode="min" />,
};

export const LiveNodeInspector: Story = {
  render: () => (
    <RunInspectionRightPanelPlayground initialRunId="run-passed" initialNodeId={DEPLOY_NODE_ID} context="live" />
  ),
};

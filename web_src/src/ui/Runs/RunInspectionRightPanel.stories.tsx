import type { Meta, StoryObj } from "@storybook/react-vite";
import { X } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import { MemoryRouter } from "react-router-dom";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { LiveCanvasSidebarRow } from "@/components/CanvasToolSidebar/LiveCanvasSidebarRow";
import { RunsTabListView } from "@/components/CanvasToolSidebar/RunsTabListView";
import { useRunFilters } from "@/components/CanvasToolSidebar/useRunFilters";
import { TimeAgo } from "@/components/TimeAgo";
import { buildNodeMap, buildRunPresentation } from "./runPresentation";
import { AccordionNodeList } from "./storybooks/accordionParts";
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
 * 40%-width panel from the right (the canvas shrinks to make room) that lists
 * the run's steps as an accordion; expanding a step unrolls its detail boxes
 * (summary / payload / runtime config) inline in the same panel.
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
          onToggleStatus={filters.toggleStatus}
          onClearStatuses={filters.clearStatuses}
          onToggleTrigger={filters.toggleTrigger}
          onClearTriggers={filters.clearTriggers}
        />
      </div>
    </div>
  );
}

function RunStepsRightPanel({
  canvasId,
  run,
  workflowNodes,
  expandedNodeId,
  onToggleNode,
  onClose,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  onClose: () => void;
}) {
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [run, nodeMap]);

  return (
    <aside className="flex h-full w-2/5 min-w-0 shrink-0 flex-col border-l border-border bg-white">
      <div className="flex shrink-0 items-start justify-between gap-2 border-b border-b-slate-950/10 px-4 py-3">
        <div className="min-w-0">
          <p className="truncate text-[13px] font-semibold text-gray-900">{presentation.title}</p>
          {run.createdAt ? (
            <span className="text-xs text-gray-500">
              <TimeAgo date={run.createdAt} />
            </span>
          ) : null}
        </div>
        <button
          type="button"
          aria-label="Close run inspection"
          onClick={onClose}
          className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-200 hover:text-slate-700"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto">
        <AccordionNodeList
          canvasId={canvasId}
          run={run}
          workflowNodes={workflowNodes}
          expandedNodeId={expandedNodeId}
          onToggleNode={onToggleNode}
        />
      </div>
    </aside>
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
}: {
  initialRunId?: string | null;
  initialNodeId?: string | null;
}) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(initialRunId);
  const [expandedNodeId, setExpandedNodeId] = useState<string | null>(initialNodeId);

  const selectedRun = useMemo(() => mockRuns.find((run) => run.id === selectedRunId) ?? null, [selectedRunId]);

  const handleSelectRun = useCallback((runId: string) => {
    setSelectedRunId(runId);
    setExpandedNodeId(null);
  }, []);

  const handleSelectLiveCanvas = useCallback(() => {
    setSelectedRunId(null);
    setExpandedNodeId(null);
  }, []);

  const toggleNode = useCallback((nodeId: string) => {
    setExpandedNodeId((current) => (current === nodeId ? null : nodeId));
  }, []);

  const selectNode = useCallback((nodeId: string) => setExpandedNodeId(nodeId), []);

  return (
    <div className="flex h-screen min-h-0 bg-white">
      <CanvasRunsSidebar isOpen>
        <RunsListSidebar
          runs={mockRuns}
          workflowNodes={mockWorkflowNodes}
          selectedRunId={selectedRunId}
          onSelectRun={handleSelectRun}
          onSelectLiveCanvas={handleSelectLiveCanvas}
        />
      </CanvasRunsSidebar>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        {selectedRun ? (
          <RunCanvas run={selectedRun} selectedNodeId={expandedNodeId} onSelectNode={selectNode} />
        ) : (
          <CanvasPlaceholder />
        )}
      </div>

      {selectedRun ? (
        <RunStepsRightPanel
          canvasId={RUNS_STORY_CANVAS_ID}
          run={selectedRun}
          workflowNodes={mockWorkflowNodes}
          expandedNodeId={expandedNodeId}
          onToggleNode={toggleNode}
          onClose={handleSelectLiveCanvas}
        />
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

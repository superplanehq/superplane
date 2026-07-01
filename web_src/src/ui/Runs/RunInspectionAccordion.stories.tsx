import type { Meta, StoryObj } from "@storybook/react-vite";
import { ArrowLeft } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { MemoryRouter } from "react-router-dom";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { RunsTabListView } from "@/components/CanvasToolSidebar/RunsTabListView";
import { useRunFilters } from "@/components/CanvasToolSidebar/useRunFilters";
import { TimeAgo } from "@/components/TimeAgo";
import { cn } from "@/lib/utils";
import { buildNodeMap, buildRunPresentation } from "./runPresentation";
import { RunCanvas } from "./storybooks/RunCanvas";
import { AccordionNodeList } from "./storybooks/accordionParts";
import {
  DEPLOY_NODE_ID,
  RUNS_STORY_CANVAS_ID,
  RunsStorySeed,
  mockFailedRun,
  mockRuns,
  mockWorkflowNodes,
} from "./storybooks/fixtures";

function AccordionRunDetailPanel({
  canvasId,
  run,
  workflowNodes,
  expandedNodeId,
  onToggleNode,
  onBack,
}: {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  onBack: () => void;
}) {
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [run, nodeMap]);

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <div className="flex h-9 min-w-0 shrink-0 items-center pl-3 pr-1">
        <button
          type="button"
          onClick={onBack}
          className="flex shrink-0 items-center gap-1 text-[13px] font-medium text-gray-500 hover:text-gray-800"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Runs
        </button>
      </div>

      <div className="shrink-0 border-b border-b-slate-950/10 px-3 py-3">
        <p className="truncate text-[13px] font-semibold text-gray-900">{presentation.title}</p>
        {run.createdAt ? (
          <span className="text-xs text-gray-500">
            <TimeAgo date={run.createdAt} />
          </span>
        ) : null}
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
    </div>
  );
}

function AccordionRunsTabPanel({
  canvasId,
  runs,
  workflowNodes,
  selectedRunId,
  onSelectRun,
  expandedNodeId,
  onToggleNode,
}: {
  canvasId: string;
  runs: CanvasesCanvasRun[];
  workflowNodes: ComponentsNode[];
  selectedRunId: string | null;
  onSelectRun: (runId: string) => void;
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
}) {
  const [view, setView] = useState<"list" | "detail">(selectedRunId ? "detail" : "list");
  const filters = useRunFilters({ runs, workflowNodes, componentIconMap: {} });
  const scrollRef = useRef<HTMLDivElement>(null);

  const selectedRun = useMemo(() => runs.find((run) => run.id === selectedRunId) ?? null, [runs, selectedRunId]);
  const isDetail = view === "detail" && !!selectedRun;

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
        <RunsTabListView
          isActive={!isDetail}
          scrollRef={scrollRef}
          onScroll={() => {}}
          runs={runs}
          filteredRuns={filters.filteredRuns}
          orderedRuns={filters.orderedRuns}
          selectedRunId={selectedRunId}
          onSelectRun={(runId) => {
            onSelectRun(runId);
            setView("detail");
          }}
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

        <div
          className={cn(
            "absolute inset-0 flex min-h-0 min-w-0 flex-col overflow-hidden bg-white transition-transform duration-300 ease-in-out",
            isDetail ? "translate-x-0" : "translate-x-full",
            isDetail ? "pointer-events-auto" : "pointer-events-none",
          )}
        >
          {selectedRun ? (
            <AccordionRunDetailPanel
              canvasId={canvasId}
              run={selectedRun}
              workflowNodes={workflowNodes}
              expandedNodeId={expandedNodeId}
              onToggleNode={onToggleNode}
              onBack={() => setView("list")}
            />
          ) : null}
        </div>
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

function RunInspectionAccordionPlayground({
  initialRunId = null,
  initialNodeId = null,
}: {
  initialRunId?: string | null;
  initialNodeId?: string | null;
}) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(initialRunId);
  const [expandedNodeId, setExpandedNodeId] = useState<string | null>(initialNodeId);

  const selectedRun = useMemo(() => mockRuns.find((run) => run.id === selectedRunId) ?? null, [selectedRunId]);

  const toggleNode = (nodeId: string) => setExpandedNodeId((current) => (current === nodeId ? null : nodeId));

  const selectNode = (nodeId: string) => setExpandedNodeId(nodeId);

  return (
    <div className="flex h-screen min-h-0 bg-white">
      <CanvasRunsSidebar isOpen>
        <AccordionRunsTabPanel
          canvasId={RUNS_STORY_CANVAS_ID}
          runs={mockRuns}
          workflowNodes={mockWorkflowNodes}
          selectedRunId={selectedRunId}
          onSelectRun={(runId) => {
            setSelectedRunId(runId);
            setExpandedNodeId(null);
          }}
          expandedNodeId={expandedNodeId}
          onToggleNode={toggleNode}
        />
      </CanvasRunsSidebar>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
        {selectedRun ? (
          <RunCanvas run={selectedRun} selectedNodeId={expandedNodeId} onSelectNode={selectNode} />
        ) : (
          <CanvasPlaceholder />
        )}
      </div>
    </div>
  );
}

const meta = {
  title: "Runs Proto/Run Inspection (Accordion)",
  component: RunInspectionAccordionPlayground,
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
} satisfies Meta<typeof RunInspectionAccordionPlayground>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => <RunInspectionAccordionPlayground />,
};

export const NodeExpanded: Story = {
  render: () => <RunInspectionAccordionPlayground initialRunId="run-passed" initialNodeId={DEPLOY_NODE_ID} />,
};

export const FailedRun: Story = {
  render: () => <RunInspectionAccordionPlayground initialRunId={mockFailedRun.id!} />,
};

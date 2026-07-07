import type { Meta, StoryObj } from "@storybook/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Background, ReactFlow, ReactFlowProvider, type Edge, type Node } from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { CheckCircle2, CircleDashed, XCircle } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import type { CanvasesCanvasNodeExecution, CanvasesCanvasRun, SuperplaneComponentsNode } from "@/api-client";
import { CanvasRunsSidebar } from "@/components/CanvasRunsSidebar";
import { RunsTabPanel } from "@/components/CanvasToolSidebar/RunsTabPanel";
import { canvasKeys } from "@/hooks/useCanvasData";
import { cn } from "@/lib/utils";
import { RunInspectorPanel } from "@/ui/Runs/RunInspectorPanel";

import {
  allOutcomeRuns,
  canvasFixture,
  componentIconMap,
  failedRunDetail,
  passedRunDetail,
  runsFixture,
  type RunDetailExample,
} from "./__fixtures__/canvasRunsExample";

// CanvasRunsPage is a composite Storybook page that combines the runs sidebar
// (RunsTabPanel inside CanvasRunsSidebar) and the right-side RunInspectorPanel with
// real captured data from the live "Clean Code Assessment" canvas. The full
// CanvasPage isn't mounted here because ReactFlow + the dozens of inter-related
// callbacks make a story brittle without adding any new visual information; the
// canvas area is rendered as a placeholder.

const meta = {
  title: "Pages/Canvas Runs",
  parameters: {
    layout: "fullscreen",
  },
} satisfies Meta<typeof RunInspectorPanel>;

export default meta;
type Story = StoryObj<typeof meta>;

interface CanvasRunsPageStoryProps {
  initialRuns: CanvasesCanvasRun[];
  initialRunId: string | null;
  initialNodeId: string | null;
  initialOpenDetail: boolean;
  showInspector: boolean;
  detailLookup: Record<string, RunDetailExample>;
  canvasArea?: (context: { selectedNodeId: string | null }) => ReactNode;
}

function CanvasRunsPageStory({
  initialRuns,
  initialRunId,
  initialNodeId,
  initialOpenDetail,
  showInspector,
  detailLookup,
  canvasArea,
}: CanvasRunsPageStoryProps) {
  const [selectedRunId, setSelectedRunId] = useState<string | null>(initialRunId);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(initialNodeId);

  const selectedRun = useMemo(
    () => initialRuns.find((run) => run.id === selectedRunId) ?? null,
    [initialRuns, selectedRunId],
  );

  const selectedDetail = selectedRunId ? (detailLookup[selectedRunId] ?? null) : null;
  const inspectorOpen = showInspector && !!selectedDetail;

  return (
    <div className="flex h-screen min-h-0 w-full flex-col bg-slate-50">
      <div className="relative flex min-h-0 flex-1 overflow-hidden">
        <CanvasRunsSidebar isOpen>
          <RunsTabPanel
            canvasId={canvasFixture.id}
            runs={initialRuns}
            selectedRunId={selectedRunId}
            selectedRun={selectedRun}
            onSelectRun={setSelectedRunId}
            onSelectLiveCanvas={() => {
              setSelectedRunId(null);
              setSelectedNodeId(null);
            }}
            onBackToRunList={() => setSelectedRunId(null)}
            initialOpenDetail={initialOpenDetail}
            selectedNodeId={selectedNodeId}
            onSelectNode={setSelectedNodeId}
            workflowNodes={canvasFixture.nodes}
            componentIconMap={componentIconMap}
          />
        </CanvasRunsSidebar>

        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {canvasArea ? canvasArea({ selectedNodeId }) : <CanvasAreaPlaceholder />}
        </div>
        {inspectorOpen && selectedDetail ? (
          <RunInspectorPanel
            canvasId={canvasFixture.id}
            run={selectedDetail.run}
            workflowNodes={canvasFixture.nodes}
            componentIconMap={componentIconMap}
            selectedNodeId={selectedNodeId}
            onSelectNode={setSelectedNodeId}
            onClearSelectedNode={() => setSelectedNodeId(null)}
            onClose={() => setSelectedRunId(null)}
          />
        ) : null}
      </div>
    </div>
  );
}

function CanvasAreaPlaceholder() {
  return (
    <div
      className="relative flex min-h-0 flex-1 items-center justify-center overflow-hidden bg-[linear-gradient(to_right,rgba(15,23,42,0.04)_1px,transparent_1px),linear-gradient(to_bottom,rgba(15,23,42,0.04)_1px,transparent_1px)]"
      style={{ backgroundSize: "24px 24px" }}
    >
      <div className="rounded-md border border-dashed border-slate-300 bg-white/70 px-4 py-2 text-xs text-slate-500 backdrop-blur-sm">
        ReactFlow canvas (placeholder)
      </div>
    </div>
  );
}

function buildPrefilledQueryClient(details: RunDetailExample[]): QueryClient {
  // useEventExecutions stores `response.data` (the full ListEventExecutions
  // response) under canvasKeys.eventExecution(canvasId, eventId). Prefilling
  // here means the right run inspector renders
  // executions synchronously without any network call.
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: Infinity, gcTime: Infinity },
    },
  });
  for (const detail of details) {
    client.setQueryData<{ executions: CanvasesCanvasNodeExecution[] }>(
      canvasKeys.eventExecution(canvasFixture.id, detail.rootEventId),
      { executions: detail.executions },
    );
  }
  return client;
}

function StoryProviders({ children }: { children: ReactNode }) {
  const queryClient = useMemo(() => buildPrefilledQueryClient([failedRunDetail, passedRunDetail]), []);

  return (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/${canvasFixture.organizationId}/apps/${canvasFixture.id}`]}>
        <Routes>
          <Route path="/:organizationId/apps/:appId" element={children} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

const detailLookup: Record<string, RunDetailExample> = {
  [failedRunDetail.run.id ?? ""]: failedRunDetail,
  [passedRunDetail.run.id ?? ""]: passedRunDetail,
};

export const Default: Story = {
  name: "Failed run + inspector",
  render: () => (
    <StoryProviders>
      <CanvasRunsPageStory
        initialRuns={runsFixture}
        initialRunId={failedRunDetail.run.id ?? null}
        initialNodeId={failedRunDetail.nodeId}
        initialOpenDetail
        showInspector
        detailLookup={detailLookup}
      />
    </StoryProviders>
  ),
};

export const PassedRun: Story = {
  name: "Passed run + inspector",
  render: () => (
    <StoryProviders>
      <CanvasRunsPageStory
        initialRuns={runsFixture}
        initialRunId={passedRunDetail.run.id ?? null}
        initialNodeId={passedRunDetail.nodeId}
        initialOpenDetail
        showInspector
        detailLookup={detailLookup}
      />
    </StoryProviders>
  ),
};

export const RunsListOnly: Story = {
  name: "Runs list only",
  render: () => (
    <StoryProviders>
      <CanvasRunsPageStory
        initialRuns={runsFixture}
        initialRunId={null}
        initialNodeId={null}
        initialOpenDetail={false}
        showInspector={false}
        detailLookup={detailLookup}
      />
    </StoryProviders>
  ),
};

export const AllOutcomes: Story = {
  name: "All run outcomes",
  render: () => (
    <StoryProviders>
      <CanvasRunsPageStory
        initialRuns={allOutcomeRuns}
        initialRunId={null}
        initialNodeId={null}
        initialOpenDetail={false}
        showInspector={false}
        detailLookup={detailLookup}
      />
    </StoryProviders>
  ),
};

// The FullPage variant renders a real ReactFlow graph in the canvas area to
// approximate the full CanvasPage look without mounting CanvasPage itself
// (which would require re-capturing action/trigger definitions, canvas edges,
// and mocking a dozen hooks that Storybook can't module-mock the way vitest
// does). Nodes appear at their captured live positions; edges are inferred
// from the passed run's execution chain (previousExecutionId → nodeId), so
// the executed subgraph is highlighted while unrelated nodes remain visible.

interface FlowNodeData extends Record<string, unknown> {
  node: SuperplaneComponentsNode;
  status: "trigger" | "passed" | "failed" | "idle";
  isSelected: boolean;
}

function CanvasBlockNode({ data }: { data: FlowNodeData }) {
  const StatusIcon = statusIconFor(data.status);
  return (
    <div
      className={cn(
        "flex w-[170px] items-center gap-2 rounded-md border bg-white px-2.5 py-2 shadow-sm",
        data.isSelected ? "border-sky-500 ring-2 ring-sky-200" : "border-slate-200",
        data.status === "failed" && "border-red-300",
        data.status === "passed" && "border-emerald-300",
        data.status === "trigger" && "border-purple-300 bg-purple-50",
      )}
    >
      <StatusIcon
        className={cn(
          "h-4 w-4 shrink-0",
          data.status === "failed" && "text-red-500",
          data.status === "passed" && "text-emerald-500",
          data.status === "trigger" && "text-purple-500",
          data.status === "idle" && "text-slate-400",
        )}
      />
      <div className="flex min-w-0 flex-col">
        <span className="truncate text-xs font-semibold text-slate-800">{data.node.name}</span>
        <span className="truncate text-[10px] text-slate-500">{data.node.component}</span>
      </div>
    </div>
  );
}

function statusIconFor(status: FlowNodeData["status"]) {
  switch (status) {
    case "passed":
      return CheckCircle2;
    case "failed":
      return XCircle;
    case "trigger":
      return CheckCircle2;
    default:
      return CircleDashed;
  }
}

const nodeTypes = { canvasBlock: CanvasBlockNode };

const COLUMN_WIDTH = 210;
const ROW_HEIGHT = 88;

function computeExecutionDepth(
  executions: CanvasesCanvasNodeExecution[],
  executionsById: Map<string, CanvasesCanvasNodeExecution>,
  triggerNodeId: string | undefined,
): Map<string, number> {
  const depthByNode = new Map<string, number>();
  if (triggerNodeId) depthByNode.set(triggerNodeId, 0);
  const depthByExecutionId = new Map<string, number>();
  const findDepth = (execution: CanvasesCanvasNodeExecution, guard: Set<string>): number => {
    if (!execution.id) return 1;
    if (depthByExecutionId.has(execution.id)) return depthByExecutionId.get(execution.id)!;
    if (guard.has(execution.id)) return 1;
    guard.add(execution.id);
    const previous = execution.previousExecutionId ? executionsById.get(execution.previousExecutionId) : undefined;
    const depth = previous ? findDepth(previous, guard) + 1 : 1;
    depthByExecutionId.set(execution.id, depth);
    return depth;
  };
  for (const execution of executions) {
    if (!execution.nodeId) continue;
    const depth = findDepth(execution, new Set());
    const current = depthByNode.get(execution.nodeId);
    if (current === undefined || depth > current) depthByNode.set(execution.nodeId, depth);
  }
  return depthByNode;
}

function buildGraph(
  workflowNodes: SuperplaneComponentsNode[],
  detail: RunDetailExample,
  selectedNodeId: string | null,
): { nodes: Node<FlowNodeData>[]; edges: Edge[] } {
  const executionsByNode = new Map<string, CanvasesCanvasNodeExecution>();
  for (const execution of detail.executions) {
    if (execution.nodeId) executionsByNode.set(execution.nodeId, execution);
  }
  const executionsById = new Map<string, CanvasesCanvasNodeExecution>();
  for (const execution of detail.executions) {
    if (execution.id) executionsById.set(execution.id, execution);
  }
  const triggerNodeId = detail.run.rootEvent?.nodeId;

  // Live positions span ~4.5k px, which fitView shrinks to unreadable sizes in
  // the story canvas. Instead, lay executed nodes out in columns following the
  // execution chain from the trigger, and fold unrelated workflow nodes below.
  const depthByNodeId = computeExecutionDepth(detail.executions, executionsById, triggerNodeId);
  const executedNodeIds = new Set(depthByNodeId.keys());
  const usedRowsPerDepth = new Map<number, number>();
  const positions = new Map<string, { x: number; y: number }>();
  for (const [nodeId, depth] of depthByNodeId.entries()) {
    const row = usedRowsPerDepth.get(depth) ?? 0;
    usedRowsPerDepth.set(depth, row + 1);
    positions.set(nodeId, { x: depth * COLUMN_WIDTH, y: row * ROW_HEIGHT });
  }
  const maxExecutedRow = Math.max(0, ...Array.from(usedRowsPerDepth.values()));
  let unrelatedIndex = 0;
  for (const node of workflowNodes) {
    if (!node.id || executedNodeIds.has(node.id)) continue;
    const column = unrelatedIndex % 5;
    const row = Math.floor(unrelatedIndex / 5);
    positions.set(node.id, {
      x: column * COLUMN_WIDTH,
      y: (maxExecutedRow + 1 + row) * ROW_HEIGHT + 48,
    });
    unrelatedIndex += 1;
  }

  const nodes: Node<FlowNodeData>[] = workflowNodes.map((node) => {
    const execution = node.id ? executionsByNode.get(node.id) : undefined;
    const isTrigger = node.id === triggerNodeId;
    const status: FlowNodeData["status"] = isTrigger
      ? "trigger"
      : execution?.result === "RESULT_FAILED"
        ? "failed"
        : execution?.result === "RESULT_PASSED"
          ? "passed"
          : "idle";
    const position = (node.id && positions.get(node.id)) ?? { x: 0, y: 0 };
    return {
      id: node.id ?? "",
      type: "canvasBlock",
      position,
      data: { node, status, isSelected: node.id === selectedNodeId },
    };
  });

  const edges: Edge[] = [];
  if (triggerNodeId) {
    // Connect the trigger to every "first" executed node (i.e. executions
    // whose previousExecutionId doesn't resolve to another execution).
    for (const execution of detail.executions) {
      const previousExec = execution.previousExecutionId
        ? executionsById.get(execution.previousExecutionId)
        : undefined;
      if (!previousExec && execution.nodeId && execution.nodeId !== triggerNodeId) {
        edges.push({
          id: `trigger-${execution.nodeId}`,
          source: triggerNodeId,
          target: execution.nodeId,
          animated: false,
        });
      }
    }
  }
  for (const execution of detail.executions) {
    if (!execution.previousExecutionId || !execution.nodeId) continue;
    const previous = executionsById.get(execution.previousExecutionId);
    if (!previous?.nodeId || previous.nodeId === execution.nodeId) continue;
    edges.push({
      id: `${previous.nodeId}-${execution.nodeId}`,
      source: previous.nodeId,
      target: execution.nodeId,
      animated: execution.result === "RESULT_FAILED",
    });
  }
  return { nodes, edges };
}

function CanvasGraphView({ detail, selectedNodeId }: { detail: RunDetailExample; selectedNodeId: string | null }) {
  const { nodes, edges } = useMemo(
    () => buildGraph(canvasFixture.nodes, detail, selectedNodeId),
    [detail, selectedNodeId],
  );

  return (
    <div className="relative min-h-0 flex-1 bg-slate-50">
      <ReactFlowProvider>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.15, maxZoom: 1 }}
          minZoom={0.25}
          maxZoom={1.5}
          proOptions={{ hideAttribution: true }}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
        >
          <Background gap={24} size={1} />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}

export const FullPage: Story = {
  name: "Full page with canvas graph",
  render: () => (
    <StoryProviders>
      <CanvasRunsPageStory
        initialRuns={runsFixture}
        initialRunId={passedRunDetail.run.id ?? null}
        initialNodeId={passedRunDetail.nodeId}
        initialOpenDetail
        showInspector
        detailLookup={detailLookup}
        canvasArea={({ selectedNodeId }) => (
          <CanvasGraphView detail={passedRunDetail} selectedNodeId={selectedNodeId} />
        )}
      />
    </StoryProviders>
  ),
};

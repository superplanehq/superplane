import { ArrowLeft, Link as LinkIcon } from "lucide-react";
import { useMemo } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { buildRunPresentation, buildNodeMap } from "@/ui/Runs/runPresentation";
import { buildExecutionChain } from "@/ui/Runs/runNodeDetailModel";
import { toast } from "sonner";
import { RunExecutionNodeRow } from "./RunExecutionNodeRow";

export interface RunDetailPanelProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  componentIconMap: Record<string, string>;
  selectedNodeId: string | null;
  onSelectNode: (nodeId: string) => void;
  onBack: () => void;
}

export function RunDetailPanel({
  canvasId,
  run,
  workflowNodes,
  componentIconMap,
  selectedNodeId,
  onSelectNode,
  onBack,
}: RunDetailPanelProps) {
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [run, nodeMap]);

  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const triggerNodeId = run.rootEvent?.nodeId;

  const executionChain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);

  const copyRunLink = () => {
    const url = new URL(window.location.href);
    url.searchParams.set("view", "runs");
    url.searchParams.set("run", run.id || "");
    navigator.clipboard.writeText(url.toString());
    toast.success("Run link copied");
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="run-detail-panel">
      <div className="flex shrink-0 items-center gap-2 px-3 pt-4 pb-1.5">
        <button
          type="button"
          onClick={onBack}
          className="flex shrink-0 items-center gap-1 text-xs font-medium text-gray-500 hover:text-gray-800"
          data-testid="run-detail-back"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Runs
        </button>
      </div>

      <div className="shrink-0 border-b border-b-slate-950/10 px-3 py-3">
        <p className="truncate text-[13px] font-semibold text-gray-900">{presentation.title}</p>
        {run.createdAt ? (
          <div className="mt-1 flex items-center gap-1">
            <span className="text-xs text-gray-500">
              <TimeAgo date={run.createdAt} />
            </span>
            <button
              type="button"
              title="Copy link to run"
              className="shrink-0 rounded p-0.5 text-gray-500 hover:bg-gray-200 hover:text-gray-600"
              onClick={copyRunLink}
            >
              <LinkIcon className="h-3 w-3" />
            </button>
          </div>
        ) : null}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto" data-testid="run-detail-node-list">
        {executionsQuery.isLoading ? (
          <p className="px-3 py-4 text-xs text-gray-400">Loading nodes...</p>
        ) : executionChain.length === 0 ? (
          <p className="px-3 py-4 text-xs text-gray-400">No executed nodes in this run.</p>
        ) : (
          executionChain.map((nodeId) => {
            const isTrigger = nodeId === triggerNodeId;
            const workflowNode = workflowNodes.find((node) => node.id === nodeId);
            const execution = executions.find((item) => item.nodeId === nodeId);

            return (
              <RunExecutionNodeRow
                key={nodeId}
                nodeId={nodeId}
                workflowNode={workflowNode}
                componentIconMap={componentIconMap}
                execution={execution}
                isTrigger={isTrigger}
                isSelected={selectedNodeId === nodeId}
                onSelect={onSelectNode}
              />
            );
          })
        )}
      </div>
    </div>
  );
}

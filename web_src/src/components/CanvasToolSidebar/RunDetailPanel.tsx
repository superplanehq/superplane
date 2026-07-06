import { ArrowLeft, ChevronLeft, ChevronRight, Link as LinkIcon } from "lucide-react";
import { useMemo } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { Timestamp } from "@/components/Timestamp";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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
  newerRunId?: string | null;
  olderRunId?: string | null;
  canNavigateOlder?: boolean;
  onNavigateRun?: (runId: string) => void;
  onNavigateOlder?: () => void;
}

export function RunDetailPanel({
  canvasId,
  run,
  workflowNodes,
  componentIconMap,
  selectedNodeId,
  onSelectNode,
  onBack,
  newerRunId = null,
  olderRunId = null,
  canNavigateOlder = false,
  onNavigateRun,
  onNavigateOlder,
}: RunDetailPanelProps) {
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [run, nodeMap]);

  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const triggerNodeId = run.rootEvent?.nodeId;

  const executionChain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);

  const copyRunLink = async () => {
    const url = new URL(window.location.href);
    url.searchParams.delete("view");
    url.searchParams.set("run", run.id || "");
    try {
      await navigator.clipboard.writeText(url.toString());
      toast.success("Run link copied");
    } catch {
      toast.error("Failed to copy run link");
    }
  };

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden" data-testid="run-detail-panel">
      <div className="flex h-9 min-w-0 shrink-0 items-stretch justify-between pl-3 pr-1">
        <div className="flex min-w-0 flex-1 items-center">
          <button
            type="button"
            onClick={onBack}
            className="flex shrink-0 items-center gap-1 rounded-md px-1 py-0.5 text-[13px] font-medium text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
            data-testid="run-detail-back"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Runs
          </button>
        </div>
        {onNavigateRun ? (
          <div className="flex shrink-0 items-stretch">
            <div className="flex items-center px-1">
              <Tooltip>
                <TooltipTrigger asChild>
                  <span>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0"
                      disabled={!newerRunId}
                      aria-label="Newer Run"
                      data-testid="run-detail-newer"
                      onClick={() => newerRunId && onNavigateRun(newerRunId)}
                    >
                      <ChevronLeft className="h-3.5 w-3.5" />
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent side="top">Newer Run</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0"
                      disabled={!canNavigateOlder}
                      aria-label="Older Run"
                      data-testid="run-detail-older"
                      onClick={() => {
                        if (olderRunId) {
                          onNavigateRun?.(olderRunId);
                          return;
                        }
                        onNavigateOlder?.();
                      }}
                    >
                      <ChevronRight className="h-3.5 w-3.5" />
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent side="top">Older Run</TooltipContent>
              </Tooltip>
            </div>
          </div>
        ) : null}
      </div>

      <div className="shrink-0 border-b border-b-slate-950/10 px-3 py-3">
        <p className="truncate text-[13px] font-semibold text-gray-900">{presentation.title}</p>
        {run.createdAt ? (
          <div className="mt-1 flex items-center gap-1">
            <span className="text-xs text-gray-500">
              <Timestamp date={run.createdAt} display="relative" relativeStyle="abbreviated" />
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

      <div className="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto" data-testid="run-detail-node-list">
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

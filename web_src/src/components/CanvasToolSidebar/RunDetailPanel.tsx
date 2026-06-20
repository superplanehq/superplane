import { ArrowLeft, ChevronLeft, ChevronRight, Link as LinkIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useEventExecutions, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
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
  const [expandedNodeIds, setExpandedNodeIds] = useState<Set<string>>(new Set());
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [run, nodeMap]);

  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const triggerNodeId = run.rootEvent?.nodeId;
  const childRunsQuery = useInfiniteCanvasRuns(
    canvasId,
    {
      parentRunId: run.id || undefined,
    },
    !!run.id,
  );

  const executionChain = useMemo(() => buildExecutionChain(executions, triggerNodeId), [executions, triggerNodeId]);
  const childRunsByNodeId = useMemo(() => {
    const byNodeId = new Map<string, CanvasesCanvasRun[]>();
    const executionNodeByID = new Map(
      (run.executions || [])
        .filter((execution): execution is { id: string; nodeId: string } => !!execution.id && !!execution.nodeId)
        .map((execution) => [execution.id, execution.nodeId]),
    );
    const pages = childRunsQuery.data?.pages || [];
    const seen = new Set<string>();

    for (const childRun of pages.flatMap((page) => page?.runs || [])) {
      if (!childRun.id || seen.has(childRun.id)) {
        continue;
      }

      seen.add(childRun.id);

      const parentNodeId = childRun.spawnedByExecutionId
        ? executionNodeByID.get(childRun.spawnedByExecutionId)
        : run.rootEvent?.nodeId;
      if (!parentNodeId) {
        continue;
      }

      const existingRuns = byNodeId.get(parentNodeId) || [];
      existingRuns.push(childRun);
      byNodeId.set(parentNodeId, existingRuns);
    }

    return byNodeId;
  }, [childRunsQuery.data?.pages, run.executions, run.rootEvent?.nodeId]);

  const toggleNodeSubRuns = (nodeId: string) => {
    setExpandedNodeIds((current) => {
      const next = new Set(current);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  };

  useEffect(() => {
    setExpandedNodeIds(new Set());
  }, [run.id]);

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
            className="flex shrink-0 items-center gap-1 text-[13px] font-medium text-gray-500 hover:text-gray-800"
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
            const childRuns = childRunsByNodeId.get(nodeId) || [];
            const hasSubRuns = childRuns.length > 0;
            const isSubRunsExpanded = expandedNodeIds.has(nodeId);

            return (
              <div key={nodeId}>
                <RunExecutionNodeRow
                  nodeId={nodeId}
                  workflowNode={workflowNode}
                  componentIconMap={componentIconMap}
                  execution={execution}
                  isTrigger={isTrigger}
                  isSelected={selectedNodeId === nodeId}
                  onSelect={onSelectNode}
                  hasSubRuns={hasSubRuns}
                  isSubRunsExpanded={isSubRunsExpanded}
                  onToggleSubRuns={hasSubRuns ? toggleNodeSubRuns : undefined}
                />
                {isSubRunsExpanded ? (
                  <div
                    className="space-y-1 border-b border-b-slate-950/10 bg-gray-50 px-3 py-2"
                    data-testid="run-detail-sub-run-group"
                  >
                    {childRuns.map((childRun) => {
                      const childRunPresentation = buildRunPresentation(childRun, nodeMap);
                      return (
                        <button
                          key={childRun.id}
                          type="button"
                          className="flex w-full items-center justify-between rounded border border-slate-950/10 bg-white px-2 py-1.5 text-left hover:bg-gray-50"
                          onClick={() => childRun.id && onNavigateRun?.(childRun.id)}
                          disabled={!onNavigateRun}
                          data-testid="run-detail-sub-run-row"
                        >
                          <span className="min-w-0 truncate pr-2 text-xs font-medium text-gray-700">
                            {childRunPresentation.title}
                          </span>
                          <span className="shrink-0 text-[10px] text-gray-400">
                            {childRun.createdAt ? <TimeAgo date={childRun.createdAt} /> : "now"}
                          </span>
                        </button>
                      );
                    })}
                    {childRunsQuery.hasNextPage ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="h-7 px-2 text-xs"
                        onClick={() => childRunsQuery.fetchNextPage()}
                        disabled={childRunsQuery.isFetchingNextPage}
                      >
                        {childRunsQuery.isFetchingNextPage ? "Loading..." : "Load more sub-runs"}
                      </Button>
                    ) : null}
                  </div>
                ) : null}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

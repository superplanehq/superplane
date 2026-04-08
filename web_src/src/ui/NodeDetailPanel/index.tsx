import type { CanvasesCanvasNodeExecution, CanvasesDescribeRunResponse, ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Button } from "@/components/ui/button";
import { cn, flattenObject } from "@/lib/utils";
import { ChevronLeft, ChevronRight, X } from "lucide-react";
import { useMemo } from "react";

interface NodeDetailPanelProps {
  nodeId: string;
  runData: CanvasesDescribeRunResponse;
  workflowNodes?: ComponentsNode[];
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
  onOpenInRunView?: (eventId: string) => void;
  isRunView?: boolean;
}

function ExecutionStatusBadge({ execution }: { execution: CanvasesCanvasNodeExecution }) {
  const state = execution.state;
  const result = execution.result;

  let label: string;
  let color: string;

  if (state === "STATE_PENDING") {
    label = "Pending";
    color = "bg-slate-200 text-slate-700";
  } else if (state === "STATE_STARTED") {
    label = "Running";
    color = "bg-blue-100 text-blue-700";
  } else if (result === "RESULT_PASSED") {
    label = "Passed";
    color = "bg-emerald-100 text-emerald-700";
  } else if (result === "RESULT_FAILED") {
    label = "Failed";
    color = "bg-red-100 text-red-700";
  } else if (result === "RESULT_CANCELLED") {
    label = "Cancelled";
    color = "bg-gray-200 text-gray-600";
  } else {
    label = "Unknown";
    color = "bg-gray-100 text-gray-500";
  }

  return (
    <span className={cn("inline-flex items-center rounded px-1.5 py-0.5 text-[11px] font-semibold uppercase", color)}>
      {label}
    </span>
  );
}

function DataSection({ title, data }: { title: string; data: Record<string, unknown> }) {
  const entries = Object.entries(data);
  if (entries.length === 0) return null;

  return (
    <div className="border-t border-slate-100 px-4 py-3">
      <h4 className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-gray-400">{title}</h4>
      <dl className="space-y-1.5">
        {entries.map(([key, value]) => (
          <div key={key} className="flex gap-2">
            <dt className="shrink-0 text-xs text-gray-500">{key}:</dt>
            <dd className="min-w-0 break-all text-xs text-gray-800">
              {typeof value === "object" ? JSON.stringify(value, null, 2) : String(value ?? "")}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

export function NodeDetailPanel({
  nodeId,
  runData,
  workflowNodes,
  onClose,
  onNavigateNode,
  isRunView,
}: NodeDetailPanelProps) {
  const executions = runData.executions || [];

  const nodeExecution = useMemo(() => executions.find((e) => e.nodeId === nodeId), [executions, nodeId]);

  const executionChain = useMemo(() => {
    const chain: string[] = [];
    const visited = new Set<string>();
    for (const exec of executions) {
      if (exec.nodeId && !visited.has(exec.nodeId)) {
        visited.add(exec.nodeId);
        chain.push(exec.nodeId);
      }
    }
    return chain;
  }, [executions]);

  const currentIndex = executionChain.indexOf(nodeId);
  const prevNodeId = currentIndex > 0 ? executionChain[currentIndex - 1] : null;
  const nextNodeId = currentIndex < executionChain.length - 1 ? executionChain[currentIndex + 1] : null;

  const nodeName = useMemo(() => {
    const node = workflowNodes?.find((n) => n.id === nodeId);
    return node?.name || nodeId;
  }, [workflowNodes, nodeId]);

  const configData = useMemo(() => {
    if (!nodeExecution?.configuration) return {};
    return flattenObject(nodeExecution.configuration);
  }, [nodeExecution?.configuration]);

  const outputData = useMemo(() => {
    if (!nodeExecution?.outputs) return {};
    return flattenObject(nodeExecution.outputs);
  }, [nodeExecution?.outputs]);

  const metadataEntries = useMemo(() => {
    if (!nodeExecution?.metadata) return {};
    return flattenObject(nodeExecution.metadata);
  }, [nodeExecution?.metadata]);

  return (
    <div className="absolute right-0 top-0 z-30 flex h-full w-80 flex-col border-l border-slate-200 bg-white shadow-lg">
      <div className="flex shrink-0 items-center justify-between border-b border-slate-200 px-4 py-2.5">
        <div className="flex items-center gap-2 min-w-0">
          <h3 className="truncate text-sm font-semibold text-gray-900">{nodeName}</h3>
          {nodeExecution ? <ExecutionStatusBadge execution={nodeExecution} /> : null}
        </div>
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {nodeExecution ? (
          <>
            <div className="px-4 py-3">
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <span>Execution #{nodeExecution.id?.slice(0, 8)}</span>
                {nodeExecution.createdAt ? (
                  <>
                    <span className="text-gray-300">·</span>
                    <TimeAgo date={nodeExecution.createdAt} />
                  </>
                ) : null}
              </div>
            </div>

            {nodeExecution.resultMessage &&
            (nodeExecution.resultReason === "RESULT_REASON_ERROR" || nodeExecution.result === "RESULT_FAILED") ? (
              <div className="border-t border-red-100 bg-red-50 px-4 py-3">
                <h4 className="mb-1 text-[11px] font-semibold uppercase tracking-wider text-red-500">Error</h4>
                <p className="text-xs text-red-700 break-all">{nodeExecution.resultMessage}</p>
              </div>
            ) : null}

            <DataSection title="Output" data={outputData} />
            <DataSection title="Configuration" data={configData} />
            <DataSection title="Metadata" data={metadataEntries} />
          </>
        ) : (
          <div className="px-4 py-8 text-center text-xs text-gray-400">
            No execution data for this node in this run.
          </div>
        )}
      </div>

      {isRunView && onNavigateNode ? (
        <div className="flex shrink-0 items-center justify-between border-t border-slate-200 px-4 py-2">
          <Button
            variant="ghost"
            size="sm"
            className="h-7 gap-1 text-xs"
            onClick={() => prevNodeId && onNavigateNode(prevNodeId)}
            disabled={!prevNodeId}
          >
            <ChevronLeft className="h-3.5 w-3.5" />
            Previous
          </Button>
          <span className="text-xs text-gray-400">
            {currentIndex >= 0 ? `${currentIndex + 1} / ${executionChain.length}` : ""}
          </span>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 gap-1 text-xs"
            onClick={() => nextNodeId && onNavigateNode(nextNodeId)}
            disabled={!nextNodeId}
          >
            Next
            <ChevronRight className="h-3.5 w-3.5" />
          </Button>
        </div>
      ) : null}
    </div>
  );
}

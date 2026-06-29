import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { RunNodeDetailContent } from "./RunNodeDetailContent";

interface RunNodeDetailModalProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  nodeId: string;
  workflowNodes?: ComponentsNode[];
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
}

/** @deprecated Use RunNodeDetailPane in runs mode. Kept for tests and backward compatibility. */
export function RunNodeDetailModal({
  canvasId,
  run,
  nodeId,
  workflowNodes = [],
  onClose,
  onNavigateNode,
}: RunNodeDetailModalProps) {
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = executionsQuery.data?.executions || [];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="flex max-h-[80vh] w-full max-w-3xl flex-col overflow-hidden rounded-lg border border-slate-200 bg-white shadow-2xl"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <RunNodeDetailContent
          run={run}
          nodeId={nodeId}
          workflowNodes={workflowNodes}
          executions={executions}
          isExecutionsLoading={executionsQuery.isLoading}
          onClose={onClose}
          onNavigateNode={onNavigateNode}
          testId="run-node-detail-modal"
        />
      </div>
    </div>
  );
}

import { Link as LinkIcon, X } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { AccordionNodeList } from "./RunStepAccordion";
import { RUN_STATUS_META, buildNodeMap, buildRunPresentation, getRunStatus } from "./runPresentation";

export interface RunPanelProps {
  canvasId: string;
  run: CanvasesCanvasRun | null;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  onClose: () => void;
  /** Tooltip for the close button, e.g. "Back to live canvas" (run inspection) or "Close" (live). */
  closeLabel?: string;
}

function CloseButton({ onClose, closeLabel }: { onClose: () => void; closeLabel: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={closeLabel}
          data-testid="run-panel-close"
          onClick={onClose}
          className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-700"
        >
          <X className="h-4 w-4" />
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{closeLabel}</TooltipContent>
    </Tooltip>
  );
}

/** Run-focused header: status pill, run title, relative time, copy link, and close button. */
function RunPanelHeader({
  run,
  workflowNodes,
  onClose,
  closeLabel,
}: {
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  onClose: () => void;
  closeLabel: string;
}) {
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [run, nodeMap]);
  const status = getRunStatus(run);
  const statusMeta = RUN_STATUS_META[status];

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
    <div className="flex shrink-0 items-start justify-between gap-2 border-b border-slate-950/10 px-3 py-2.5">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span
            className={cn(
              "inline-flex shrink-0 items-center gap-1 rounded-full px-1.5 py-0.5 text-[11px] font-medium ring-1 ring-inset",
              statusMeta.badgeClassName,
            )}
          >
            <span className={cn("h-1.5 w-1.5 rounded-full", statusMeta.dotClassName)} />
            {statusMeta.label}
          </span>
          <p className="truncate text-[13px] font-semibold text-gray-900">{presentation.title}</p>
        </div>
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
      <CloseButton onClose={onClose} closeLabel={closeLabel} />
    </div>
  );
}

/**
 * Dedicated run panel: a run-focused header above the run-step accordion. Used
 * for both run inspection and the live node inspector. The outer width / border
 * wrapper is provided by the caller (CanvasPage).
 */
export function RunPanel({
  canvasId,
  run,
  workflowNodes,
  componentIconMap = {},
  expandedNodeId,
  onToggleNode,
  onClose,
  closeLabel = "Close",
}: RunPanelProps) {
  if (!run) {
    return (
      <div className="flex h-full min-h-0 flex-col overflow-hidden bg-white">
        <div className="flex shrink-0 items-center justify-end border-b border-slate-950/10 px-3 py-2.5">
          <CloseButton onClose={onClose} closeLabel={closeLabel} />
        </div>
        <div className="flex min-h-0 flex-1 items-center justify-center px-6 py-16 text-center">
          <p className="text-xs text-gray-400">This node has not run yet.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-white">
      <RunPanelHeader run={run} workflowNodes={workflowNodes} onClose={onClose} closeLabel={closeLabel} />
      <div className="min-h-0 flex-1 overflow-y-auto" data-testid="run-panel-step-list">
        <AccordionNodeList
          canvasId={canvasId}
          run={run}
          workflowNodes={workflowNodes}
          componentIconMap={componentIconMap}
          expandedNodeId={expandedNodeId}
          onToggleNode={onToggleNode}
        />
      </div>
    </div>
  );
}

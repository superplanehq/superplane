import { Link as LinkIcon, X } from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { TimeAgo } from "@/components/TimeAgo";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { AccordionNodeList } from "./RunStepAccordion";
import { buildNodeMap, buildRunPresentation } from "./runPresentation";

export interface RunInspectionSidePanelProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  expandedNodeId: string | null;
  onToggleNode: (nodeId: string) => void;
  onClose: () => void;
}

export function RunInspectionSidePanel({
  canvasId,
  run,
  workflowNodes,
  componentIconMap = {},
  expandedNodeId,
  onToggleNode,
  onClose,
}: RunInspectionSidePanelProps) {
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [run, nodeMap]);

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
    <div
      className="flex h-full min-h-0 w-2/5 shrink-0 flex-col overflow-hidden border-l border-slate-950/10 bg-white"
      data-testid="run-inspection-side-panel"
    >
      <div className="flex shrink-0 items-start justify-between gap-2 border-b border-slate-950/10 px-3 py-2.5">
        <div className="min-w-0 flex-1">
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
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              aria-label="Close run inspection"
              data-testid="run-inspection-side-panel-close"
              onClick={onClose}
              className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-700"
            >
              <X className="h-4 w-4" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top">Close</TooltipContent>
        </Tooltip>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto" data-testid="run-inspection-step-list">
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

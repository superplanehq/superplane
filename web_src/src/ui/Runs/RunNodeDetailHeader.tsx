import { ChevronLeft, ChevronRight, X } from "lucide-react";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";

export interface RunNodeDetailHeaderProps {
  nodeName: string;
  workflowNode?: ComponentsNode;
  componentIconMap: Record<string, string>;
  previousNodeId: string | null;
  nextNodeId: string | null;
  onClose: () => void;
  onNavigateNode?: (nodeId: string) => void;
}

export function RunNodeDetailHeader({
  nodeName,
  workflowNode,
  componentIconMap,
  previousNodeId,
  nextNodeId,
  onClose,
  onNavigateNode,
}: RunNodeDetailHeaderProps) {
  return (
    <div className="flex h-9 shrink-0 items-stretch justify-between border-b border-slate-200 pl-3 dark:border-gray-800/70">
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <div className="flex min-w-0 items-center gap-1.5">
          <RunNodeIcon
            componentName={workflowNode?.component}
            iconSlug={workflowNode?.component ? componentIconMap[workflowNode.component] : undefined}
            alt={nodeName}
            size={RUN_NODE_ICON_SIZE}
            className="h-3.5 w-3.5 shrink-0 text-gray-800 dark:text-gray-100"
          />
          <h3 className="truncate text-[13px] font-medium text-gray-900 dark:text-gray-100">{nodeName}</h3>
        </div>
      </div>
      <div className="flex shrink-0 items-stretch">
        {onNavigateNode ? (
          <div className="flex items-center px-1">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0"
              disabled={!previousNodeId}
              aria-label="Previous node in run"
              onClick={() => previousNodeId && onNavigateNode(previousNodeId)}
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 w-6 p-0"
              disabled={!nextNodeId}
              aria-label="Next node in run"
              onClick={() => nextNodeId && onNavigateNode(nextNodeId)}
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </Button>
          </div>
        ) : null}
        <div aria-hidden className="w-px self-stretch bg-slate-200 dark:bg-gray-800/50" />
        <div className="flex items-center px-1">
          <Button type="button" variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onClose}>
            <X className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
    </div>
  );
}

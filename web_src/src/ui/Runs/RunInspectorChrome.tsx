import { ChevronsRight } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export function RunInspectorChrome({ onClose }: { onClose: () => void }) {
  return (
    <div className="flex shrink-0 items-center justify-between gap-2 border-b border-slate-950/10 px-2 py-1.5 dark:border-gray-800">
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="flex h-7 w-7 items-center justify-center rounded text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
            data-testid="run-panel-close"
          >
            <ChevronsRight className="h-4 w-4" />
          </button>
        </TooltipTrigger>
        <TooltipContent side="bottom">Close</TooltipContent>
      </Tooltip>
    </div>
  );
}

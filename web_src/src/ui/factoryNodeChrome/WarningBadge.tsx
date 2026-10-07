import { CircleAlert } from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

export function WarningBadge({ text }: { text: string }) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div
            data-testid="node-warning-badge"
            className="absolute -top-8 left-0 flex h-6 w-6 cursor-pointer items-center justify-center rounded-full bg-orange-500 dark:bg-orange-400/25 dark:ring-1 dark:ring-orange-400/50"
          >
            <CircleAlert className="h-4 w-4 text-white dark:text-orange-300" />
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <p className="max-w-xs text-sm">{text}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

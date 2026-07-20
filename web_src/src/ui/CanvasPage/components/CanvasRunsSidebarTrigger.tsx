import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { Button as UIButton } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Rabbit } from "lucide-react";
import {
  canvasSidebarToggleActiveClassName,
  canvasSidebarToggleInactiveClassName,
} from "./canvasSidebarToggleClassNames";

export type CanvasRunsSidebarTriggerProps = {
  runsSidebarState: CanvasRunsSidebarState;
};

export function CanvasRunsSidebarTrigger({ runsSidebarState }: CanvasRunsSidebarTriggerProps) {
  const { showRunsSidebarToggle, isRunsSidebarOpen, handleRunsSidebarToggle, runningRunsCount = 0 } = runsSidebarState;
  const label = "Toggle Runs";
  const hasRunningRuns = runningRunsCount > 0;

  if (!showRunsSidebarToggle) {
    return null;
  }

  return (
    <Tooltip delayDuration={350}>
      <TooltipTrigger asChild>
        <UIButton
          type="button"
          variant="ghost"
          size="icon-xs"
          className={cn(
            "relative size-7 rounded-full border-0 shadow-none transition-colors",
            isRunsSidebarOpen ? canvasSidebarToggleActiveClassName : canvasSidebarToggleInactiveClassName,
          )}
          aria-label={hasRunningRuns ? `${label}, ${runningRunsCount} running` : label}
          aria-pressed={isRunsSidebarOpen}
          data-testid="canvas-runs-sidebar-toggle"
          onClick={handleRunsSidebarToggle}
        >
          <Rabbit className="size-4 shrink-0" />
          {hasRunningRuns ? (
            <span
              className="absolute -right-3.5 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-blue-500 px-1 text-[10px] font-semibold leading-none text-white ring-2 ring-blue-500/20 before:absolute before:inset-[-2px] before:rounded-full before:bg-blue-500/15 before:content-[''] before:animate-ping after:absolute after:inset-0 after:rounded-full after:ring-2 after:ring-white dark:ring-blue-400/20 dark:before:bg-blue-400/15 dark:after:ring-gray-900"
              aria-hidden="true"
            >
              {runningRunsCount}
            </span>
          ) : null}
        </UIButton>
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={2}>
        {label}
      </TooltipContent>
    </Tooltip>
  );
}

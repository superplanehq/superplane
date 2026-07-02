import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { Button as UIButton } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { History } from "lucide-react";

export type CanvasRunsSidebarTriggerProps = {
  runsSidebarState: CanvasRunsSidebarState;
};

export function CanvasRunsSidebarTrigger({ runsSidebarState }: CanvasRunsSidebarTriggerProps) {
  const { showRunsSidebarToggle, isRunsSidebarOpen, handleRunsSidebarToggle, runningRunsCount = 0 } = runsSidebarState;
  const label = "Toggle Runs";
  const hasRunning = runningRunsCount > 0;

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
            "relative size-7 overflow-visible rounded-full border-0 shadow-none transition-colors",
            isRunsSidebarOpen
              ? "bg-slate-200 text-foreground hover:bg-slate-200 focus-visible:bg-slate-200"
              : "bg-slate-100 text-slate-500 hover:bg-slate-100 hover:text-foreground focus-visible:bg-slate-100",
          )}
          aria-label={hasRunning ? `${label} (${runningRunsCount} running)` : label}
          aria-pressed={isRunsSidebarOpen}
          data-testid="canvas-runs-sidebar-toggle"
          onClick={handleRunsSidebarToggle}
        >
          <History className="size-3.5 shrink-0" />
          {hasRunning ? (
            <span
              className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center"
              data-testid="canvas-runs-running-badge"
            >
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75" />
              <span className="relative inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-blue-500 px-1 text-[10px] font-semibold leading-none text-white">
                {runningRunsCount > 9 ? "9+" : runningRunsCount}
              </span>
            </span>
          ) : null}
        </UIButton>
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={2}>
        {hasRunning ? `${label} · ${runningRunsCount} running` : label}
      </TooltipContent>
    </Tooltip>
  );
}

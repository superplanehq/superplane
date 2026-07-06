import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { Button as UIButton } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PanelLeft, PanelLeftDashed } from "lucide-react";

export type CanvasRunsSidebarTriggerProps = {
  runsSidebarState: CanvasRunsSidebarState;
};

export function CanvasRunsSidebarTrigger({ runsSidebarState }: CanvasRunsSidebarTriggerProps) {
  const { showRunsSidebarToggle, isRunsSidebarOpen, handleRunsSidebarToggle } = runsSidebarState;
  const label = "Toggle Runs";

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
            "size-7 rounded-full border-0 shadow-none transition-colors",
            isRunsSidebarOpen
              ? "bg-slate-300 text-slate-950 hover:bg-slate-300 hover:text-slate-950 focus-visible:bg-slate-300 dark:bg-gray-300 dark:text-gray-950 dark:hover:bg-gray-300 dark:hover:text-gray-950 dark:focus-visible:bg-gray-300"
              : "bg-slate-100 text-slate-500 hover:bg-slate-100 hover:text-foreground focus-visible:bg-slate-100 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700 dark:focus-visible:bg-gray-700",
          )}
          aria-label={label}
          aria-pressed={isRunsSidebarOpen}
          data-testid="canvas-runs-sidebar-toggle"
          onClick={handleRunsSidebarToggle}
        >
          {isRunsSidebarOpen ? (
            <PanelLeft className="size-3.5 shrink-0" />
          ) : (
            <PanelLeftDashed className="size-3.5 shrink-0" />
          )}
        </UIButton>
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={2}>
        {label}
      </TooltipContent>
    </Tooltip>
  );
}

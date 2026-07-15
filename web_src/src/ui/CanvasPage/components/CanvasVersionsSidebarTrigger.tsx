import type { CanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";
import { Button as UIButton } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { History } from "lucide-react";
import {
  canvasSidebarToggleActiveClassName,
  canvasSidebarToggleInactiveClassName,
} from "./canvasSidebarToggleClassNames";

export type CanvasVersionsSidebarTriggerProps = {
  versionsSidebarState: CanvasVersionsSidebarState;
};

export function CanvasVersionsSidebarTrigger({ versionsSidebarState }: CanvasVersionsSidebarTriggerProps) {
  const { showVersionsSidebarToggle, isVersionsSidebarOpen, handleVersionsSidebarToggle } = versionsSidebarState;
  const label = "Toggle Versions";

  if (!showVersionsSidebarToggle) {
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
            isVersionsSidebarOpen ? canvasSidebarToggleActiveClassName : canvasSidebarToggleInactiveClassName,
          )}
          aria-label={label}
          aria-pressed={isVersionsSidebarOpen}
          data-testid="canvas-versions-sidebar-toggle"
          onClick={handleVersionsSidebarToggle}
        >
          <History className="size-4 shrink-0" />
        </UIButton>
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={2}>
        {label}
      </TooltipContent>
    </Tooltip>
  );
}

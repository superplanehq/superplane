import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PanelLeft, PanelLeftDashed } from "lucide-react";

export type CanvasToolSidebarTriggerProps = {
  toolSidebarState: CanvasToolSidebarState;
};

export function CanvasToolSidebarTrigger({ toolSidebarState }: CanvasToolSidebarTriggerProps) {
  const { showToolSidebarToggle, isToolSidebarOpen, handleToolSidebarToggle } = toolSidebarState;

  if (!showToolSidebarToggle) {
    return null;
  }

  const label = isToolSidebarOpen ? "Close sidebar" : "Open sidebar";
  const tooltip = isToolSidebarOpen ? "Close sidebar" : "Open sidebar";

  return (
    <div className="relative z-10 -ml-2 flex shrink-0 items-center">
      <Tooltip>
        <TooltipTrigger asChild>
          <UIButton
            type="button"
            variant="ghost"
            size="icon-xs"
            className="rounded-md border-0 p-0 shadow-none transition-colors focus-visible:bg-slate-100 text-slate-900 hover:bg-slate-100 hover:text-slate-900"
            aria-label={label}
            aria-pressed={isToolSidebarOpen}
            onClick={handleToolSidebarToggle}
          >
            {isToolSidebarOpen ? (
              <PanelLeft className="size-3.5 shrink-0" />
            ) : (
              <PanelLeftDashed className="size-3.5 shrink-0" />
            )}
          </UIButton>
        </TooltipTrigger>
        <TooltipContent side="right" sideOffset={8}>
          {tooltip}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

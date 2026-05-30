import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { Button as UIButton } from "@/components/ui/button";
import { useShortcutLabel } from "@/hooks/useShortcutLabel";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PanelLeft, PanelLeftDashed } from "lucide-react";

export type CanvasToolSidebarTriggerProps = {
  toolSidebarState: CanvasToolSidebarState;
};

export function CanvasToolSidebarTrigger({ toolSidebarState }: CanvasToolSidebarTriggerProps) {
  const { showToolSidebarToggle, isToolSidebarOpen, handleToolSidebarToggle } = toolSidebarState;
  const shortcutLabel = useShortcutLabel("B");
  const label = `Toggle Sidebar (${shortcutLabel})`;

  if (!showToolSidebarToggle) {
    return null;
  }

  return (
    <div className="relative z-10 -ml-2 flex shrink-0 items-center">
      <Tooltip delayDuration={350}>
        <TooltipTrigger asChild>
          <UIButton
            type="button"
            variant="ghost"
            size="icon-xs"
            className="rounded-md border-0 p-0 shadow-none transition-colors focus-visible:bg-slate-100 text-slate-900 hover:bg-slate-100 hover:text-slate-900"
            aria-label={label}
            aria-pressed={isToolSidebarOpen}
            data-testid="canvas-tool-sidebar-toggle"
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
          {label}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { Button as UIButton } from "@/components/ui/button";
import { useShortcutLabel } from "@/hooks/useShortcutLabel";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Sparkle, Sparkles } from "lucide-react";

export type CanvasToolSidebarTriggerProps = {
  toolSidebarState: CanvasToolSidebarState;
};

export function CanvasToolSidebarTrigger({ toolSidebarState }: CanvasToolSidebarTriggerProps) {
  const { showToolSidebarToggle, isToolSidebarOpen, handleToolSidebarToggle } = toolSidebarState;
  const shortcutLabel = useShortcutLabel("B");
  const label = `Toggle Agent (${shortcutLabel})`;

  if (!showToolSidebarToggle) {
    return null;
  }

  return (
    <div className="relative z-10 flex h-7 shrink-0 items-center">
      <Tooltip delayDuration={350}>
        <TooltipTrigger asChild>
          <UIButton
            type="button"
            variant="ghost"
            size={null}
            className={cn(
              "h-7 min-h-7 gap-1.5 rounded-full border-0 py-1 pl-2.5 pr-4 text-[13px] shadow-none transition-colors",
              isToolSidebarOpen
                ? "bg-violet-100 hover:bg-violet-100 focus-visible:bg-violet-100"
                : "bg-slate-100 text-slate-500 hover:bg-slate-100 hover:text-foreground focus-visible:bg-slate-100",
            )}
            aria-label={label}
            aria-pressed={isToolSidebarOpen}
            data-testid="canvas-tool-sidebar-toggle"
            onClick={handleToolSidebarToggle}
          >
            {isToolSidebarOpen ? (
              <Sparkles className="size-3.5 shrink-0 text-violet-600" />
            ) : (
              <Sparkle className="size-3.5 shrink-0" />
            )}
            <span className={cn("text-[13px] font-medium whitespace-nowrap", isToolSidebarOpen && "text-violet-600")}>
              Agent
            </span>
          </UIButton>
        </TooltipTrigger>
        <TooltipContent side="right" sideOffset={2}>
          {label}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

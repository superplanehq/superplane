import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Sparkles } from "lucide-react";

export type AgentSidebarTriggerProps = {
  showAgentSidebarToggle?: boolean;
  isAgentSidebarOpen?: boolean;
  onToggleAgentSidebar?: () => void;
};

export function AgentSidebarTrigger({
  showAgentSidebarToggle,
  isAgentSidebarOpen,
  onToggleAgentSidebar,
}: AgentSidebarTriggerProps) {
  return (
    <div className="relative z-10 flex shrink-0 items-center">
      {showAgentSidebarToggle && onToggleAgentSidebar && !isAgentSidebarOpen ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="relative inline-flex">
              <UIButton
                type="button"
                variant="outline"
                size="icon"
                className="h-8 w-8 bg-white border-slate-300"
                aria-label="Open SuperPlane Agent"
                onClick={onToggleAgentSidebar}
              >
                <Sparkles className="h-3 w-3 text-slate-700" />
              </UIButton>
            </span>
          </TooltipTrigger>
          <TooltipContent side="right" sideOffset={8}>
            Open Agent
          </TooltipContent>
        </Tooltip>
      ) : null}
    </div>
  );
}

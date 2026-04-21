import type { AgentState } from "@/components/AgentSidebar/useAgentState";
import { Button as UIButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { Sparkles } from "lucide-react";

export type AgentSidebarTriggerProps = {
  agentState: AgentState;
};

export function AgentSidebarTrigger({ agentState }: AgentSidebarTriggerProps) {
  const { showAgentSidebarToggle, isAgentSidebarOpen, handleAgentSidebarToggle } = agentState;

  return (
    <div className="relative z-10 flex shrink-0 items-center">
      {showAgentSidebarToggle ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="relative inline-flex">
              <UIButton
                type="button"
                variant="outline"
                size="sm"
                aria-pressed={isAgentSidebarOpen}
                aria-label={isAgentSidebarOpen ? "Close SuperPlane Agent" : "Open SuperPlane Agent"}
                onClick={handleAgentSidebarToggle}
                className={cn(
                  "border transition-colors",
                  isAgentSidebarOpen
                    ? "border-violet-300 bg-violet-100 text-violet-700 hover:bg-violet-100/90 hover:text-violet-800"
                    : "border-slate-300 bg-white text-slate-700 hover:bg-slate-50",
                )}
              >
                <Sparkles
                  className={cn("size-3.5", isAgentSidebarOpen ? "text-violet-600" : "text-slate-700")}
                  aria-hidden
                />
              </UIButton>
            </span>
          </TooltipTrigger>
          <TooltipContent side="right" sideOffset={8}>
            {isAgentSidebarOpen ? "Close Agent" : "Open Agent"}
          </TooltipContent>
        </Tooltip>
      ) : null}
    </div>
  );
}

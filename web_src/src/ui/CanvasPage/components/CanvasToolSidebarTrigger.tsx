import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { requestAgentComposerSend } from "@/components/AgentSidebar/composerPrefill";
import { Button as UIButton } from "@/components/ui/button";
import { useShortcutLabel } from "@/hooks/useShortcutLabel";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Sparkle, Sparkles } from "lucide-react";
import { useEffect, useState } from "react";
import { useParams } from "react-router";
import { AgentSuggestionsHoverCard, type AgentSuggestion } from "./AgentSuggestionsHoverCard";
import {
  canvasSidebarToggleActiveClassName,
  canvasSidebarToggleInactiveClassName,
} from "./canvasSidebarToggleClassNames";
import { useAgentSuggestionDismissals } from "./useAgentSuggestionDismissals";

export type CanvasToolSidebarTriggerProps = {
  toolSidebarState: CanvasToolSidebarState;
  /** Short improvement prompts shown on hover when the Agent control has suggestions. */
  agentSuggestions?: AgentSuggestion[];
  onSelectAgentSuggestion?: (suggestion: AgentSuggestion) => void;
};

export function CanvasToolSidebarTrigger({
  toolSidebarState,
  agentSuggestions = [],
  onSelectAgentSuggestion,
}: CanvasToolSidebarTriggerProps) {
  const {
    organizationId = "",
    appId,
    canvasId: canvasIdParam,
    workflowId,
  } = useParams<{
    organizationId?: string;
    appId?: string;
    canvasId?: string;
    workflowId?: string;
  }>();
  const canvasId = appId || canvasIdParam || workflowId || "";
  const { showToolSidebarToggle, isToolSidebarOpen, handleToolSidebarToggle, openToolSidebar } = toolSidebarState;
  const shortcutLabel = useShortcutLabel("B");
  const { visibleSuggestions, dismissSuggestion } = useAgentSuggestionDismissals(
    organizationId,
    canvasId,
    agentSuggestions,
  );
  // Sticky on first paint until outside click / Escape; choosing a suggestion removes only that row.
  const [cardOpen, setCardOpen] = useState(true);

  // SPA navigation reuses this trigger; reopen the card for the active canvas.
  useEffect(() => {
    setCardOpen(true);
  }, [canvasId]);

  const suggestionCount = visibleSuggestions.length;
  const label =
    suggestionCount > 0
      ? `Toggle Agent (${shortcutLabel}), ${suggestionCount} suggested improvements`
      : `Toggle Agent (${shortcutLabel})`;

  if (!showToolSidebarToggle) {
    return null;
  }

  const handleSelectSuggestion = (suggestion: AgentSuggestion) => {
    onSelectAgentSuggestion?.(suggestion);
    // Queue the send before opening so it is pending when the composer mounts.
    requestAgentComposerSend(canvasId, suggestion.prompt);
    openToolSidebar();
    dismissSuggestion(suggestion.id);
    // Keep the card open while other suggestions remain (sidebar open can steal focus).
    if (visibleSuggestions.some((item) => item.id !== suggestion.id)) {
      setCardOpen(true);
    }
  };

  const button = (
    <UIButton
      type="button"
      variant="ghost"
      size={null}
      className={cn(
        "h-7 min-h-7 gap-1.5 rounded-full border-0 py-1 pl-2.5 pr-4 text-[13px] shadow-none transition-colors",
        isToolSidebarOpen ? canvasSidebarToggleActiveClassName : canvasSidebarToggleInactiveClassName,
      )}
      aria-label={label}
      aria-pressed={isToolSidebarOpen}
      data-testid="canvas-tool-sidebar-toggle"
      onClick={handleToolSidebarToggle}
    >
      {isToolSidebarOpen ? (
        <Sparkles className="size-3.5 shrink-0 text-violet-800" />
      ) : (
        <Sparkle className="size-3.5 shrink-0" />
      )}
      <span className={cn("text-[13px] font-medium whitespace-nowrap", isToolSidebarOpen && "text-violet-800")}>
        Agent
      </span>
    </UIButton>
  );

  if (suggestionCount > 0) {
    return (
      <div className="relative z-10 flex h-7 shrink-0 items-center">
        <AgentSuggestionsHoverCard
          suggestions={visibleSuggestions}
          open={cardOpen}
          onOpenChange={setCardOpen}
          onSelectSuggestion={handleSelectSuggestion}
        >
          {button}
        </AgentSuggestionsHoverCard>
      </div>
    );
  }

  return (
    <div className="relative z-10 flex h-7 shrink-0 items-center">
      <Tooltip delayDuration={350}>
        <TooltipTrigger asChild>{button}</TooltipTrigger>
        <TooltipContent side="right" sideOffset={2}>
          {label}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

import { useMemo } from "react";
import { getAgentSuggestions } from "@/lib/agentSuggestionsContext";
import type { AgentSuggestion } from "@/ui/CanvasPage";

/** Post-install Agent improvement suggestions for the current canvas. */
export function useAppPageAgentSuggestions(canvasId: string | undefined): AgentSuggestion[] {
  return useMemo(() => {
    if (!canvasId) return [];
    return getAgentSuggestions(canvasId);
  }, [canvasId]);
}

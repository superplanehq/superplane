import { useCanvasPreference, useUpdateCanvasPreference } from "@/hooks/useCanvasData";
import { useEffect, useMemo, useState } from "react";
import type { AgentSuggestion } from "./AgentSuggestionsHoverCard";

/** Tracks DB-backed Agent suggestion dismissals for the current canvas. */
export function useAgentSuggestionDismissals(
  organizationId: string,
  canvasId: string,
  agentSuggestions: AgentSuggestion[],
) {
  const preferenceQueryEnabled = !!organizationId && !!canvasId;
  const { data: canvasPreference, isPending: preferencePending } = useCanvasPreference(
    organizationId,
    canvasId,
    preferenceQueryEnabled,
  );
  const { mutate: updateCanvasPreference } = useUpdateCanvasPreference(organizationId);
  const [dismissedIds, setDismissedIds] = useState<ReadonlySet<string>>(() => new Set());
  const dismissedFromServerKey = (canvasPreference?.dismissedAgentSuggestionIds ?? []).join("\0");
  // Avoid flashing already-dismissed suggestions before the DB preference loads.
  const preferenceReady = !preferenceQueryEnabled || !preferencePending;

  useEffect(() => {
    if (!preferenceReady) return;
    setDismissedIds(new Set(dismissedFromServerKey ? dismissedFromServerKey.split("\0") : []));
  }, [canvasId, dismissedFromServerKey, preferenceReady]);

  const visibleSuggestions = useMemo(() => {
    if (!preferenceReady) return [];
    return agentSuggestions.filter((suggestion) => !dismissedIds.has(suggestion.id));
  }, [agentSuggestions, dismissedIds, preferenceReady]);

  const dismissSuggestion = (suggestionId: string) => {
    if (canvasId && organizationId) {
      updateCanvasPreference({ canvasId, dismissAgentSuggestionId: suggestionId });
    }
    setDismissedIds((prev) => new Set(prev).add(suggestionId));
  };

  return { visibleSuggestions, dismissSuggestion };
}

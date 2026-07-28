import { useCanvasPreference, useUpdateCanvasPreference } from "@/hooks/useCanvasData";
import { useEffect, useMemo, useState } from "react";
import type { AgentSuggestion } from "./AgentSuggestionsHoverCard";

/** Tracks DB-backed Agent suggestion dismissals for the current canvas. */
export function useAgentSuggestionDismissals(
  organizationId: string,
  canvasId: string,
  agentSuggestions: AgentSuggestion[],
) {
  const preferenceQueryEnabled = !!organizationId && !!canvasId && agentSuggestions.length > 0;
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
    setDismissedIds(new Set());
  }, [canvasId]);

  // Union server dismissals into local state so a partial cache write cannot undismiss.
  useEffect(() => {
    if (!preferenceReady) return;
    const fromServer = dismissedFromServerKey ? dismissedFromServerKey.split("\0") : [];
    setDismissedIds((prev) => new Set([...prev, ...fromServer]));
  }, [canvasId, dismissedFromServerKey, preferenceReady]);

  const visibleSuggestions = useMemo(() => {
    if (!preferenceReady) return [];
    return agentSuggestions.filter((suggestion) => !dismissedIds.has(suggestion.id));
  }, [agentSuggestions, dismissedIds, preferenceReady]);

  const dismissSuggestion = (suggestionId: string) => {
    setDismissedIds((prev) => new Set(prev).add(suggestionId));
    if (!canvasId || !organizationId) return;

    updateCanvasPreference(
      { canvasId, dismissAgentSuggestionId: suggestionId },
      {
        onError: () => {
          setDismissedIds((prev) => {
            const next = new Set(prev);
            next.delete(suggestionId);
            return next;
          });
        },
      },
    );
  };

  return { visibleSuggestions, dismissSuggestion };
}

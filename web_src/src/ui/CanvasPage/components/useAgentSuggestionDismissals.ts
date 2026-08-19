import { useCanvas, useUpdateCanvas } from "@/hooks/useCanvasData";
import { useEffect, useMemo, useState } from "react";
import type { AgentSuggestion } from "./AgentSuggestionsHoverCard";

/** Tracks canvas-scoped Agent suggestion dismissals for the current app. */
export function useAgentSuggestionDismissals(
  organizationId: string,
  canvasId: string,
  agentSuggestions: AgentSuggestion[],
) {
  const canvasQueryEnabled = !!organizationId && !!canvasId && agentSuggestions.length > 0;
  const { data: canvas, isPending: canvasPending } = useCanvas(organizationId, canvasId, {
    enabled: canvasQueryEnabled,
    staleTime: 30_000,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
  });
  const { mutate: updateCanvas } = useUpdateCanvas(organizationId, canvasId);
  const [dismissedIds, setDismissedIds] = useState<ReadonlySet<string>>(() => new Set());
  const dismissedFromServerKey = (canvas?.metadata?.dismissedAgentSuggestionIds ?? []).join("\0");
  // Avoid flashing already-dismissed suggestions before the canvas loads.
  const dismissalsReady = !canvasQueryEnabled || !canvasPending;

  useEffect(() => {
    setDismissedIds(new Set());
  }, [canvasId]);

  // Union server dismissals into local state so a partial cache write cannot undismiss.
  useEffect(() => {
    if (!dismissalsReady) return;
    const fromServer = dismissedFromServerKey ? dismissedFromServerKey.split("\0") : [];
    setDismissedIds((prev) => new Set([...prev, ...fromServer]));
  }, [canvasId, dismissedFromServerKey, dismissalsReady]);

  const visibleSuggestions = useMemo(() => {
    if (!dismissalsReady) return [];
    return agentSuggestions.filter((suggestion) => !dismissedIds.has(suggestion.id));
  }, [agentSuggestions, dismissedIds, dismissalsReady]);

  const dismissSuggestion = (suggestionId: string) => {
    setDismissedIds((prev) => new Set(prev).add(suggestionId));
    if (!canvasId || !organizationId) return;

    updateCanvas(
      { dismissAgentSuggestionId: suggestionId },
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

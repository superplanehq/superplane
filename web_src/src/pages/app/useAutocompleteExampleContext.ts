import type { CanvasesCanvas } from "@/api-client";
import { useMemo } from "react";
import { buildAutocompleteAppExample, type AutocompleteExampleContext } from "./buildAutocompleteExampleObj";

type AutocompleteExampleContextInput = Omit<AutocompleteExampleContext, "app"> & {
  canvas: CanvasesCanvas | null | undefined;
};

export function useAutocompleteExampleContext({
  canvas,
  canvasNodes,
  canvasNodesById,
  incomingNodeIdsByTargetId,
  visibleNodeExecutionsMap,
  visibleNodeEventsMap,
  allComponentsByName,
  allTriggersByName,
}: AutocompleteExampleContextInput): AutocompleteExampleContext {
  return useMemo(
    () => ({
      canvasNodes,
      canvasNodesById,
      incomingNodeIdsByTargetId,
      visibleNodeExecutionsMap,
      visibleNodeEventsMap,
      allComponentsByName,
      allTriggersByName,
      app: buildAutocompleteAppExample(canvas),
    }),
    [
      canvas,
      canvasNodes,
      canvasNodesById,
      incomingNodeIdsByTargetId,
      visibleNodeExecutionsMap,
      visibleNodeEventsMap,
      allComponentsByName,
      allTriggersByName,
    ],
  );
}

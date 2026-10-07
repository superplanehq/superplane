import { useParams } from "react-router";

import { useCanvas } from "@/hooks/useCanvasData";

export type FactoryModelScope = {
  factoryId: string | undefined;
  waitingForCanvas: boolean;
};

/** Use a known workspace id. Otherwise wait for the canvas route. */
export function preferredFactoryScope(
  explicitFactoryId: string | undefined,
  canvasScope: FactoryModelScope,
): FactoryModelScope {
  const factoryId = explicitFactoryId?.trim() ?? "";
  if (factoryId === "") {
    return canvasScope;
  }
  return { factoryId, waitingForCanvas: false };
}

/** Factory that owns the open canvas, so model lists follow that factory allowlist. */
export function useCanvasFactoryScope(organizationId: string | undefined) {
  const { appId } = useParams<{ appId?: string }>();
  const canvasQuery = useCanvas(organizationId ?? "", appId ?? "", {
    enabled: Boolean(organizationId && appId),
    staleTime: Infinity,
  });
  return {
    factoryId: canvasQuery.data?.metadata?.factoryId,
    waitingForCanvas: Boolean(appId) && canvasQuery.isPending,
  };
}

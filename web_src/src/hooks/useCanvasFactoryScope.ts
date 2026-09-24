import { useParams } from "react-router";

import { useCanvas } from "@/hooks/useCanvasData";

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

import { useParams } from "react-router";
import { isAppRouteId } from "@/lib/appPaths";

export const useCanvasId = (): string | null => {
  const { appId, automationId, canvasId } = useParams<{
    appId?: string;
    automationId?: string;
    canvasId?: string;
  }>();
  const routeAppId = automationId ?? appId;
  if (routeAppId) {
    return isAppRouteId(routeAppId) ? routeAppId : null;
  }
  return canvasId || null;
};

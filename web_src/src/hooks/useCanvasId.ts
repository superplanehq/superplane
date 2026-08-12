import { useParams } from "react-router";
import { isAppRouteId } from "@/lib/appPaths";

export const useCanvasId = (): string | null => {
  const { appId, canvasId } = useParams<{ appId?: string; canvasId?: string }>();
  if (appId) {
    return isAppRouteId(appId) ? appId : null;
  }
  return canvasId || null;
};

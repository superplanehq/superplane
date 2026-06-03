import { useParams } from "react-router-dom";

export const useCanvasId = (): string | null => {
  const { appId, canvasId } = useParams<{ appId?: string; canvasId?: string }>();
  return appId || canvasId || null;
};

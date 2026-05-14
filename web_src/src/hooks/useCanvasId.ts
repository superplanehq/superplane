import { useParams } from "react-router-dom";

export const useCanvasId = (): string | null => {
  const { canvasId } = useParams<{ canvasId?: string }>();
  return canvasId || null;
};

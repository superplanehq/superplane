import {
  loadCanvasAppPreferences,
  setCanvasPinned,
  setCanvasStarred,
  type CanvasAppPreferences,
} from "@/lib/canvasAppPreferences";
import { useCallback, useEffect, useState } from "react";

export function useCanvasAppPreferences(organizationId: string, accountId?: string) {
  const [preferences, setPreferences] = useState<CanvasAppPreferences>(() =>
    loadCanvasAppPreferences(organizationId, accountId),
  );

  useEffect(() => {
    setPreferences(loadCanvasAppPreferences(organizationId, accountId));
  }, [accountId, organizationId]);

  const pinCanvas = useCallback(
    (canvasId: string, pinned: boolean) => {
      setPreferences(setCanvasPinned(organizationId, accountId, canvasId, pinned));
    },
    [accountId, organizationId],
  );

  const starCanvas = useCallback(
    (canvasId: string, starred: boolean) => {
      setPreferences(setCanvasStarred(organizationId, accountId, canvasId, starred));
    },
    [accountId, organizationId],
  );

  return { preferences, pinCanvas, starCanvas };
}

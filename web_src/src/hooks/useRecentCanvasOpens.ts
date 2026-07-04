import { loadRecentCanvasOpens, recordRecentCanvasOpen, type RecentCanvasOpen } from "@/lib/recentCanvasOpens";
import { useCallback, useEffect, useState } from "react";

export function useRecentCanvasOpens(organizationId: string) {
  const [recentOpens, setRecentOpens] = useState<RecentCanvasOpen[]>(() => loadRecentCanvasOpens(organizationId));

  useEffect(() => {
    setRecentOpens(loadRecentCanvasOpens(organizationId));
  }, [organizationId]);

  const recordOpen = useCallback(
    (canvasId: string) => {
      if (!organizationId || !canvasId) {
        return;
      }

      setRecentOpens(recordRecentCanvasOpen(organizationId, canvasId));
    },
    [organizationId],
  );

  return { recentOpens, recordOpen };
}

import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { canvasKeys } from "@/hooks/useCanvasData";

export type RefreshLatestLiveCanvasDataOptions = {
  liveVersionId?: string;
};

export function useRefreshLatestLiveCanvasData(
  organizationId: string | undefined,
  canvasId: string | undefined,
  liveCanvasVersionId: string | undefined,
) {
  const queryClient = useQueryClient();

  return useCallback(
    async (options?: RefreshLatestLiveCanvasDataOptions) => {
      if (!organizationId || !canvasId) {
        return;
      }

      const liveVersionId = options?.liveVersionId ?? liveCanvasVersionId;
      const invalidations: Array<Promise<unknown>> = [
        queryClient.invalidateQueries({
          queryKey: canvasKeys.detail(organizationId, canvasId),
          refetchType: "all",
        }),
        queryClient.invalidateQueries({
          queryKey: canvasKeys.versionHistory(canvasId),
          refetchType: "all",
        }),
        queryClient.invalidateQueries({
          queryKey: canvasKeys.canvasStaging(canvasId),
          refetchType: "all",
        }),
      ];

      if (liveVersionId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: canvasKeys.console(canvasId, liveVersionId),
            exact: true,
            refetchType: "all",
          }),
        );
      }

      invalidations.push(
        queryClient.invalidateQueries({
          queryKey: canvasKeys.console(canvasId, undefined),
          exact: true,
          refetchType: "all",
        }),
      );

      await Promise.all(invalidations);
    },
    [organizationId, canvasId, queryClient, liveCanvasVersionId],
  );
}

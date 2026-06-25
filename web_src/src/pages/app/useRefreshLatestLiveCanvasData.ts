import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { canvasKeys } from "@/hooks/useCanvasData";

export type RefreshLatestLiveCanvasDataOptions = {
  liveVersionId?: string;
  skipDraftBranchRefetch?: boolean;
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
      ];

      if (!options?.skipDraftBranchRefetch) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: canvasKeys.draftBranches(canvasId),
            refetchType: "all",
          }),
        );
      }

      if (liveVersionId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: canvasKeys.console(canvasId, liveVersionId),
            exact: true,
            refetchType: "all",
          }),
        );
      }

      // Live view reads console.yaml via the version-less "live" cache key while
      // edit mode uses the draft version id; refresh both after publish/exit.
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

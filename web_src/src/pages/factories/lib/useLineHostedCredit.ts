import { useQueries } from "@tanstack/react-query";

import { canvasesDescribeCanvas } from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { lineUsesHostedCredit, type LineRunnerNode } from "./lineHostedCredit";

/**
 * Whether the line's agent runners spend hosted credit.
 * Undefined while the canvases are still loading or a describe call failed.
 * An empty app list means the line has no agent app, so the run does not spend hosted credit.
 */
export function useLineUsesHostedCredit(
  organizationId: string | undefined,
  appIds: readonly string[],
): boolean | undefined {
  const org = organizationId ?? "";
  const queries = useQueries({
    queries: appIds.map((appId) => ({
      queryKey: canvasKeys.detail(org, appId),
      queryFn: async () => {
        const response = await canvasesDescribeCanvas(
          withOrganizationHeader({
            path: { id: appId },
          }),
        );
        return response.data?.canvas;
      },
      enabled: Boolean(org && appId),
      staleTime: 30 * 1000,
    })),
  });

  if (appIds.length === 0) {
    return false;
  }
  if (queries.some((query) => query.isPending || query.isError)) {
    return undefined;
  }

  const nodes = queries.flatMap((query) => canvasRunnerNodes(query.data));
  return lineUsesHostedCredit(nodes);
}

function canvasRunnerNodes(canvas: { spec?: { nodes?: LineRunnerNode[] } } | undefined): LineRunnerNode[] {
  return canvas?.spec?.nodes ?? [];
}
